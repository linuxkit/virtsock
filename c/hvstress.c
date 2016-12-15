/*
 * A simple Hyper-V sockets stress test.
 *
 * This program uses a configurable number of client threads which all
 * open a connection to a server and then transfer a random amount of
 * data to the server which echos the data back.
 */
#include "compat.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* 3049197C-9A4E-4FBF-9367-97F792F16994 */
DEFINE_GUID(MY_SERVICE_GUID,
    0x3049197c, 0x9a4e, 0x4fbf, 0x93, 0x67, 0x97, 0xf7, 0x92, 0xf1, 0x69, 0x94);

#define SVR_BUF_LEN (3 * 4096)
#define MAX_BUF_LEN (20 * 1024 * 1024)
#define DEFAULT_CLIENT_CONN 100

static int verbose;
#define INFO(...)                                                       \
    do {                                                                \
        if (verbose) {                                                  \
            printf(__VA_ARGS__);                                        \
            fflush(stdout);                                             \
        }                                                               \
    } while (0)
#define DBG(...)                                                        \
    do {                                                                \
        if (verbose > 1) {                                              \
            printf(__VA_ARGS__);                                        \
            fflush(stdout);                                             \
        }                                                               \
    } while (0)
#define TRC(...)                                                        \
    do {                                                                \
        if (verbose > 2) {                                              \
            printf(__VA_ARGS__);                                        \
            fflush(stdout);                                             \
        }                                                               \
    } while (0)

#ifdef _MSC_VER
static WSADATA wsaData;
#endif

struct svr_args {
    SOCKET fd;
    int conn;
};

/* Handle a connection. Echo back anything sent to us and when the
 * connection is closed send a bye message.
 */
static void *handle(void *a)
{
    struct svr_args *args = a;
    uint64_t start, end, diff;
    char recvbuf[SVR_BUF_LEN];
    int recvbuflen = SVR_BUF_LEN;
    int total_bytes = 0;
    int received;
    int sent;
    int res;

    TRC("[%05d] server thread: Handle fd=%d\n", args->conn, (int)args->fd);

    start = time_ns();

    for (;;) {
        received = recv(args->fd, recvbuf, recvbuflen, 0);
        if (received == 0) {
            DBG("[%05d] Peer closed\n", args->conn);
            break;
        } else if (received == SOCKET_ERROR) {
            sockerr("recv()");
            goto out;
        }

        sent = 0;
        while (sent < received) {
            res = send(args->fd, recvbuf + sent, received - sent, 0);
            if (res == SOCKET_ERROR) {
                sockerr("send()");
                goto out;
            }
            sent += res;
        }
        total_bytes += sent;
    }

    end = time_ns();

out:
    diff = end - start;
    diff /= 1000 * 1000;
    INFO("[%05d] ECHOED: %9d Bytes in %5"PRIu64"ms\n",
         args->conn, total_bytes, diff);
    TRC("close(%d)\n", (int)args->fd);
    closesocket(args->fd);
    free(args);
    return NULL;
}


/* Server:
 * accept() in an endless loop, handle a connection at a time
 */
static int server(void)
{
    SOCKET lsock, csock;
    SOCKADDR_HV sa, sac;
    socklen_t socklen = sizeof(sac);
    struct svr_args *args;
    THREAD_HANDLE st;
    int conn = 0;
    int res;

    lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
    if (lsock == INVALID_SOCKET) {
        sockerr("socket()");
        return 1;
    }

    sa.Family = AF_HYPERV;
    sa.Reserved = 0;
    sa.VmId = HV_GUID_WILDCARD;
    sa.ServiceId = MY_SERVICE_GUID;

    res = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
    if (res == SOCKET_ERROR) {
        sockerr("bind()");
        closesocket(lsock);
        return 1;
    }

    res = listen(lsock, SOMAXCONN);
    if (res == SOCKET_ERROR) {
        sockerr("listen()");
        closesocket(lsock);
        return 1;
    }

    while(1) {
        csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
        if (csock == INVALID_SOCKET) {
            sockerr("accept()");
            closesocket(lsock);
            return 1;
        }

        DBG("Connect from: "GUID_FMT":"GUID_FMT" fd=%d\n",
            GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId), (int)csock);

        /* Spin up a new thread per connection. Not the most
         * efficient, but stops us from having to faff about with
         * worker threads and the like. */
        args = malloc(sizeof(*args));
        if (!args) {
            fprintf(stderr, "failled to malloc thread state\n");
            return 1;
        }
        args->fd = csock;
        args->conn = conn++;
        thread_create(&st, &handle, args);
        thread_detach(st);
    }
}


/* Client code
 *
 * The client sends one message of random size and expects the server
 * to echo it back. The sending is done in a separate thread so we can
 * simultaneously drain the server's replies.  Could do this in a
 * single thread with poll()/select() as well, but this keeps the code
 * simpler.
 */

/* Arguments for client threads */
struct client_args {
    THREAD_HANDLE h;
    GUID target;
    int id;
    int conns;

    int res;
};

/* Argument passed to Client send thread */
struct client_tx_args {
    SOCKET fd;
    int tosend;
    int id;
    int conn;
};


#if _MSC_VER
/* XXX On Windows, calling send with 32K causes the connection being
 * closed!!!  The MSDN pages for send() is ambiguous about this. We
 * could use getsockopt(SO_MAX_MSG_SIZE) to query the size, but
 * instead hard code this value to 8k. */
#define MAX_SND_BUF (8 * 1024)
#else
#define MAX_SND_BUF (32 * 1024)
#endif
static void *client_tx(void *a)
{
    struct client_tx_args *args = a;
    char sendbuf[MAX_SND_BUF];
    char tmp[128];
    int tosend, this_batch;
    int res;

    tosend = args->tosend;
    while (tosend) {
        this_batch = (tosend >  MAX_SND_BUF) ? MAX_SND_BUF : tosend;
        res = send(args->fd, sendbuf, this_batch, 0);
        if (res == SOCKET_ERROR) {
            snprintf(tmp, sizeof(tmp), "[%02d:%05d] send() after %d bytes",
                     args->id, args->conn, args->tosend - tosend);
            sockerr(tmp);
            goto out;
        }
        tosend -= res;
    }
    DBG("[%02d:%05d] TX: %9d bytes sent\n", args->id, args->conn, args->tosend);

out:
    return NULL;
}

/* Client code for a single connection */
static int client_one(GUID target, int id, int conn)
{
    struct client_tx_args args;
    uint64_t start, end, diff;
    THREAD_HANDLE st;
    SOCKADDR_HV sa;
    SOCKET fd;
    char *recvbuf;
    char tmp[128];
    int tosend, received = 0;
    int res;

    TRC("[%02d:%05d] start\n", id, conn);

    recvbuf = malloc(MAX_BUF_LEN);
    if (!recvbuf) {
        fprintf(stderr, "[%02d:%05d] Failed to allocate recvbuf\n", id, conn);
        return 1;
    }

    start = time_ns();

    fd = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
    if (fd == INVALID_SOCKET) {
        sockerr("socket()");
        free(recvbuf);
        return 1;
    }

    sa.Family = AF_HYPERV;
    sa.Reserved = 0;
    sa.VmId = target;
    sa.ServiceId = MY_SERVICE_GUID;

    TRC("[%02d:%05d] Connect to: "GUID_FMT":"GUID_FMT"\n",
        id, conn, GUID_ARGS(sa.VmId), GUID_ARGS(sa.ServiceId));

    res = connect(fd, (const struct sockaddr *)&sa, sizeof(sa));
    if (res == SOCKET_ERROR) {
        sockerr("connect()");
        goto out;
    }
    DBG("[%02d:%05d] Connected to: "GUID_FMT":"GUID_FMT" fd=%d\n",
        id, conn, GUID_ARGS(sa.VmId), GUID_ARGS(sa.ServiceId), (int)fd);

    if (RAND_MAX < MAX_BUF_LEN)
        tosend = (int)((1ULL * RAND_MAX + 1) * rand() + rand());
    else
        tosend = rand();

    tosend = tosend % (MAX_BUF_LEN - 1) + 1;

    DBG("[%02d:%05d] TOSEND: %d bytes\n", id, conn, tosend);
    args.fd = fd;
    args.tosend = tosend;
    args.id = id;
    args.conn = conn;
    thread_create(&st, &client_tx, &args);

    while (received < tosend) {
        res = recv(fd, recvbuf, MAX_BUF_LEN, 0);
        if (res < 0) {
            snprintf(tmp, sizeof(tmp), "[%02d:%05d] recv() after %d bytes",
                     id, conn, received);
            sockerr(tmp);
            goto thout;
        } else if (res == 0) {
            INFO("[%02d:%05d] Connection closed\n", id, conn);
            res = 1;
            goto thout;
        }
        received += res;
    }

    res = 0;

thout:
    thread_join(st);
    end = time_ns();
    diff = end - start;
    diff /= 1000 * 1000;
    INFO("[%02d:%05d] TX/RX: %9d bytes in %5"PRIu64"ms\n",
         id, conn, received, diff);
out:
    TRC("[%02d:%05d] close(%d)\n", id, conn, (int)fd);
    closesocket(fd);
    free(recvbuf);
    return res;
}

static void *client_thd(void *a)
{
    struct client_args *args = a;
    int res, i;

    for (i = 0; i < args->conns; i++) {
        res = client_one(args->target, args->id, i);
        if (res)
            break;
    }

    args->res = res;
    return args;
}

void usage(char *name)
{
    printf("%s: -s|-c <carg> [-i <conns>]\n", name);
    printf(" -s         Server mode\n");
    printf(" -c <carg>  Client mode. <carg>:\n");
    printf("   'loopback': Connect in loopback mode\n");
    printf("   'parent':   Connect to the parent partition\n");
    printf("   <guid>:     Connect to VM with GUID\n");
    printf(" -i <conns> Number connections the client makes (default %d)\n",
           DEFAULT_CLIENT_CONN);
    printf(" -p <num>   Run 'num' connections in parallel (default 1)\n");
    printf(" -r         Initialise random number generator with the time\n");
    printf(" -v         Verbose output\n");
}

int __cdecl main(int argc, char **argv)
{
    struct client_args *args;
    int opt_conns = DEFAULT_CLIENT_CONN;
    int opt_server = 0;
    int opt_par = 1;
    GUID target;
    int res = 0;
    int i;

#ifdef _MSC_VER
    /* Initialize Winsock */
    res = WSAStartup(MAKEWORD(2,2), &wsaData);
    if (res != 0) {
        fprintf(stderr, "WSAStartup() failed with error: %d\n", res);
        return 1;
    }
#endif

    /* No getopt on windows. Do some manual parsing */
    for (i = 1; i < argc; i++) {
        if (strcmp(argv[i], "-s") == 0) {
            opt_server = 1;
        } else if (strcmp(argv[i], "-c") == 0) {
            opt_server = 0;
            if (i + 1 >= argc) {
                fprintf(stderr, "-c requires an argument\n");
                usage(argv[0]);
                goto out;
            }
            if (strcmp(argv[i + 1], "loopback") == 0) {
                target = HV_GUID_LOOPBACK;
            } else if (strcmp(argv[i + 1], "parent") == 0) {
                target = HV_GUID_PARENT;
            } else {
                res = parseguid(argv[i + 1], &target);
                if (res != 0) {
                    fprintf(stderr, "failed to scan: %s\n", argv[i + 1]);
                    goto out;
                }
            }
            i++;

        } else if (strcmp(argv[i], "-i") == 0) {
            if (i + 1 >= argc) {
                fprintf(stderr, "-i requires an argument\n");
                usage(argv[0]);
                goto out;
            }
            opt_conns = atoi(argv[++i]);
        } else if (strcmp(argv[i], "-p") == 0) {
            if (i + 1 >= argc) {
                fprintf(stderr, "-p requires an argument\n");
                usage(argv[0]);
                goto out;
            }
            opt_par = atoi(argv[++i]);
        } else if (strcmp(argv[i], "-r") == 0) {
            srand(time(NULL));
        } else if (strcmp(argv[i], "-v") == 0) {
            verbose++;
        } else {
            usage(argv[0]);
            goto out;
        }
    }

    if (opt_server) {
        server();
    } else {
        args = calloc(opt_par, sizeof(*args));
        if (!args) {
            fprintf(stderr, "failed to malloc");
            res = -1;
            goto out;
        }

        /* Create threads */
        for (i = 0; i < opt_par; i++) {
            args[i].target = target;
            args[i].id = i;
            args[i].conns = opt_conns / opt_par;
            thread_create(&args[i].h, &client_thd, &args[i]);
        }

        /* Wait for threads to finish and collect return codes */
        res = 0;
        for (i = 0; i < opt_par; i++) {
            thread_join(args[i].h);
            if (args[i].res)
                fprintf(stderr, "THREAD[%d] failed with %d", i, args[i].res);
            res |= args[i].res;
        }
    }

out:
#ifdef _MSC_VER
    WSACleanup();
#endif
    return res;
}
