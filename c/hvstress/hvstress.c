/*
 * The start of a Hyper-V sockets stress test program.
 *
 * TODO:
 * - Add more concurrency
 * - Verify that the data send is the same as received.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "compat.h"

/* 3049197C-9A4E-4FBF-9367-97F792F16994 */
DEFINE_GUID(MY_SERVICE_GUID,
    0x3049197c, 0x9a4e, 0x4fbf, 0x93, 0x67, 0x97, 0xf7, 0x92, 0xf1, 0x69, 0x94);

#define SVR_BUF_LEN (3 * 4096)
#define MAX_BUF_LEN (2 * 1024 * 1024)


/* Helper macros for parsing/printing GUIDs */
#define GUID_FMT "%08x-%04hx-%04hx-%02x%02x-%02x%02x%02x%02x%02x%02x"
#define GUID_ARGS(_g)                                               \
    (_g).Data1, (_g).Data2, (_g).Data3,                             \
    (_g).Data4[0], (_g).Data4[1], (_g).Data4[2], (_g).Data4[3],     \
    (_g).Data4[4], (_g).Data4[5], (_g).Data4[6], (_g).Data4[7]
#define GUID_SARGS(_g)                                              \
    &(_g).Data1, &(_g).Data2, &(_g).Data3,                          \
    &(_g).Data4[0], &(_g).Data4[1], &(_g).Data4[2], &(_g).Data4[3], \
    &(_g).Data4[4], &(_g).Data4[5], &(_g).Data4[6], &(_g).Data4[7]


int parseguid(const char *s, GUID *g)
{
    int res;
    int p0, p1, p2, p3, p4, p5, p6, p7;

    res = sscanf(s, GUID_FMT,
                 &g->Data1, &g->Data2, &g->Data3,
                 &p0, &p1, &p2, &p3, &p4, &p5, &p6, &p7);
    if (res != 11)
        return 1;
    g->Data4[0] = p0;
    g->Data4[1] = p1;
    g->Data4[2] = p2;
    g->Data4[3] = p3;
    g->Data4[4] = p4;
    g->Data4[5] = p5;
    g->Data4[6] = p6;
    g->Data4[7] = p7;
    return 0;
}

/* Slightly different error handling between Windows and Linux */
void sockerr(const char *msg)
{
#ifdef _MSC_VER
    fprintf(stderr, "%s Error: %d\n", msg, WSAGetLastError());
#else
    fprintf(stderr, "%s Error: %d. %s", msg, errno, strerror(errno));
#endif
}

#ifdef _MSC_VER
static WSADATA wsaData;
#endif

/* Argument passed to Client send thread */
struct client_args {
    SOCKET fd;
    int tosend;
};

/* Handle a connection. Echo back anything sent to us and when the
 * connection is closed send a bye message.
 */
static void handle(SOCKET fd)
{
    char recvbuf[SVR_BUF_LEN];
    int recvbuflen = SVR_BUF_LEN;
    int total_bytes = 0;
    int received;
    int sent;
    int res;

    for (;;) {
        received = recv(fd, recvbuf, recvbuflen, 0);
        if (received == 0) {
            printf("Peer closed\n");
            break;
        } else if (received == SOCKET_ERROR) {
            sockerr("recv()");
            return;
        }

        /* No error, echo */
        /* printf("RX: %d Bytes\n", received); */

        sent = 0;
        while (sent < received) {
            res = send(fd, recvbuf + sent, received - sent, 0);
            if (sent == SOCKET_ERROR) {
                sockerr("send()");
                return;
            }
            /* printf("TX: %d Bytes\n", res); */
            sent += res;
        }
        total_bytes += sent;
    }

    printf("ECHO: %d Bytes\n", total_bytes);

    /* Dummy read to wait till other end closes */
    recv(fd, recvbuf, recvbuflen, 0);
}


/* Server:
 * accept() in an endless loop, handle a connection at a time
 */
static int server(void)
{
    SOCKET lsock = INVALID_SOCKET;
    SOCKET csock = INVALID_SOCKET;
    SOCKADDR_HV sa, sac;
    socklen_t socklen = sizeof(sac);
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

        printf("Connect from: "GUID_FMT":"GUID_FMT"\n",
               GUID_ARGS(sac.VmId), GUID_ARGS(sac.ServiceId));

        handle(csock);
        closesocket(csock);
    }
}


/* Client send function, executed in a different thread
 *
 * It sends garbage (ie whatever happens to be in memory. It mixes
 * larger with smaller send requests just because we can.
 */
#define MAX_SND_BUF (32 * 1024)
static void *client_send(void *a)
{
    char sendbuf[MAX_SND_BUF];
    struct client_args *args = a;
    int tosend, this_batch;
    int res;

    tosend = args->tosend;
    this_batch = 4;
    while (tosend) {
        res = send(args->fd, sendbuf, this_batch, 0);
        if (res == SOCKET_ERROR) {
            sockerr("send()");
            goto out;
        }
        tosend -= res;
        if (this_batch == 4)
            this_batch = (tosend >  MAX_SND_BUF) ? MAX_SND_BUF : tosend;
        else
            this_batch = 4;

    }
    printf("TX: %d bytes sent\n", res);

out:
    return NULL;
}

/* The client sends a messages, and waits for the echo before shutting
 * down the send side. It then expects a bye message from the server.
 */
static int client(GUID target)
{
    SOCKET fd = INVALID_SOCKET;
    SOCKADDR_HV sa;
    THREAD_HANDLE st;
    struct client_args args;
    char *recvbuf;
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

    printf("Connect to: "GUID_FMT":"GUID_FMT"\n",
           GUID_ARGS(sa.VmId), GUID_ARGS(sa.ServiceId));

    res = connect(fd, (const struct sockaddr *)&sa, sizeof(sa));
    if (res == SOCKET_ERROR) {
        sockerr("connect()");
        goto out;
    }

    tosend = rand();
    if (RAND_MAX < MAX_BUF_LEN)
        tosend = (double)tosend / ((long long)RAND_MAX + 1) *  (MAX_BUF_LEN - 1) + 1;
    else
        tosend = tosend % (MAX_BUF_LEN - 1) + 1;

    printf("TX: %d bytes\n", tosend);
    args.fd = fd;
    args.tosend = tosend;
    thread_create(&st, &client_send, &args);

    while (received < tosend) {
        res = recv(fd, recvbuf, MAX_BUF_LEN, 0);
        if (res < 0) {
            sockerr("recv()");
            goto out;
        } else if (res == 0) {
            printf("Connection closed\n");
            res = 1;
            goto out;
        }
        /* printf("RX: %d bytes\n", res); */
        received += res;
    }
    printf("RX: %d bytes\n", received);
    res = 0;

 out:
    closesocket(fd);
    free(recvbuf);
    return res;
}

void usage(char *name)
{
    printf("%s: -s | -c <carg>\n", name);
    printf("In client mode <carg>:\n");
    printf("<empty>:  Connect in loopback mode\n");
    printf("'parent': Connect to the parent partition\n");
    printf("<guid>:   Connect to VM with GUID\n");
}

int __cdecl main(int argc, char **argv)
{
    int res = 0;
    GUID target;
    int i;

#ifdef _MSC_VER
    // Initialize Winsock
    res = WSAStartup(MAKEWORD(2,2), &wsaData);
    if (res != 0) {
        fprintf(stderr, "WSAStartup() failed with error: %d\n", res);
        return 1;
    }
#endif

    if (argc < 2 || argc > 3 || strcmp(argv[1], "-l") == 0) {
        usage(argv[0]);
        goto out;
    }

    if (strcmp(argv[1], "-s") == 0) {
        res = server();
    } else if (strcmp(argv[1], "-c") == 0) {
        if (argc == 2) {
            target = HV_GUID_LOOPBACK;
        } else if (strcmp(argv[1], "parent") == 0) {
            target = HV_GUID_PARENT;
        } else {
            res = parseguid(argv[2], &target);
            if (res != 0) {
                fprintf(stderr, "failed to scan: %s\n", argv[2]);
                goto out;
            }
        }

        for (i = 0; i < 100; i++) {
            printf ("TEST: %d\n", i);
            res = client(target);
            if (res)
                break;
        }
    } else {
        usage(argv[0]);
        res = 1;
    }

out:
#ifdef _MSC_VER
    WSACleanup();
#endif
    return res;
}
