/*
 * The start of a Hyper-V sockets stress test program.
 *
 * TODO:
 * - Add more concurrency
 * - Verify that the data send is the same as received.
 */
#include "compat.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* 3049197C-9A4E-4FBF-9367-97F792F16994 */
DEFINE_GUID(MY_SERVICE_GUID,
    0x3049197c, 0x9a4e, 0x4fbf, 0x93, 0x67, 0x97, 0xf7, 0x92, 0xf1, 0x69, 0x94);

#define SVR_BUF_LEN (3 * 4096)
#define MAX_BUF_LEN (2 * 1024 * 1024)
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

/* Handle a connection. Echo back anything sent to us and when the
 * connection is closed send a bye message.
 */
static void *handle(void *a)
{
    SOCKET fd = *(SOCKET *)a;
    char recvbuf[SVR_BUF_LEN];
    int recvbuflen = SVR_BUF_LEN;
    int total_bytes = 0;
    int received;
    int sent;
    int res;

    TRC("server thread: Handle fd=%d\n", (int)fd);

    for (;;) {
        received = recv(fd, recvbuf, recvbuflen, 0);
        if (received == 0) {
            DBG("Peer closed\n");
            break;
        } else if (received == SOCKET_ERROR) {
            sockerr("recv()");
            goto out;
        }

        sent = 0;
        while (sent < received) {
            res = send(fd, recvbuf + sent, received - sent, 0);
            if (res == SOCKET_ERROR) {
                sockerr("send()");
                goto out;
            }
            sent += res;
        }
        total_bytes += sent;
    }

out:
    INFO("ECHOED: %d Bytes\n", total_bytes);
    free(a);
    TRC("close(%d)\n", (int)fd);
    closesocket(fd);
    return NULL;
}


/* Server:
 * accept() in an endless loop, handle a connection at a time
 */
static int server(void)
{
    SOCKET lsock, csock, *sp;
    SOCKADDR_HV sa, sac;
    socklen_t socklen = sizeof(sac);
    THREAD_HANDLE st;
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
        sp = malloc(sizeof(*sp));
        *sp = csock;
        thread_create(&st, &handle, sp);
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

/* Argument passed to Client send thread */
struct client_tx_args {
    SOCKET fd;
    int tosend;
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
            snprintf(tmp, sizeof(tmp), "send() after %d bytes",
                     args->tosend - tosend);
            sockerr(tmp);
            goto out;
        }
        tosend -= res;
    }
    DBG("TX: %d bytes sent\n", args->tosend);

out:
    return NULL;
}

/* Client code for a single connection */
static int client_one(GUID target)
{
    struct client_tx_args args;
    THREAD_HANDLE st;
    SOCKADDR_HV sa;
    SOCKET fd;
    char *recvbuf;
    char tmp[128];
    int tosend, received = 0;
    int res;

    recvbuf = malloc(MAX_BUF_LEN);
    if (!recvbuf) {
        fprintf(stderr, "Failed to allocate recvbuf\n");
        return 1;
    }

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

    TRC("Connect to: "GUID_FMT":"GUID_FMT"\n",
        GUID_ARGS(sa.VmId), GUID_ARGS(sa.ServiceId));

    res = connect(fd, (const struct sockaddr *)&sa, sizeof(sa));
    if (res == SOCKET_ERROR) {
        sockerr("connect()");
        goto out;
    }
    DBG("Connected to: "GUID_FMT":"GUID_FMT" fd=%d\n",
        GUID_ARGS(sa.VmId), GUID_ARGS(sa.ServiceId), (int)fd);

    tosend = rand();
    if (RAND_MAX < MAX_BUF_LEN)
        tosend = (int)((double)tosend / 
                       ((long long)RAND_MAX + 1) *  (MAX_BUF_LEN - 1) + 1);
    else
        tosend = tosend % (MAX_BUF_LEN - 1) + 1;

    DBG("TOSEND: %d bytes\n", tosend);
    args.fd = fd;
    args.tosend = tosend;
    thread_create(&st, &client_tx, &args);

    while (received < tosend) {
        res = recv(fd, recvbuf, MAX_BUF_LEN, 0);
        if (res < 0) {
            snprintf(tmp, sizeof(tmp), "recv() after %d bytes", received);
            sockerr(tmp);
            goto thout;
        } else if (res == 0) {
            INFO("Connection closed\n");
            res = 1;
            goto thout;
        }
        received += res;
    }
    INFO("RX: %d bytes received\n", received);
    res = 0;

thout:
    thread_join(st);
out:
    TRC("close(%d)\n", (int)fd);
    closesocket(fd);
    free(recvbuf);
    return res;
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
    printf(" -r         Initialise random number generator with the time\n");
    printf(" -v         Verbose output\n");
}

int __cdecl main(int argc, char **argv)
{
    int opt_server = 0;
    int opt_conns = DEFAULT_CLIENT_CONN;
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
        } else if (strcmp(argv[i], "-r") == 0) {
            srand(time(NULL));
        } else if (strcmp(argv[i], "-v") == 0) {
            verbose++;
        } else {
            usage(argv[0]);
            goto out;
        }
    }

    if (opt_server)
        server();
    else
        for (i = 0; i < opt_conns; i++) {
            INFO("TEST: %d\n", i);
            res = client_one(target);
            if (res)
                break;
        }

out:
#ifdef _MSC_VER
    WSACleanup();
#endif
    return res;
}
