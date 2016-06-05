/*
 * A simple Echo server and client using Hyper-V sockets
 *
 * Works on Linux and Windows (kinda)
 *
 * This was primarily written to checkout shutdown(), which turns out
 * does not work.
 */
#include "compat.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* 3049197C-9A4E-4FBF-9367-97F792F16994 */
DEFINE_GUID(MY_SERVICE_GUID,
    0x3049197c, 0x9a4e, 0x4fbf, 0x93, 0x67, 0x97, 0xf7, 0x92, 0xf1, 0x69, 0x94);

#define MY_BUFLEN 4096

#ifdef _MSC_VER
static WSADATA wsaData;
#endif

/* Handle a connection. Echo back anything sent to us and when the
 * connection is closed send a bye message.
 */
static void handle(SOCKET fd)
{
    char recvbuf[MY_BUFLEN];
    int recvbuflen = MY_BUFLEN;
    const char *byebuf = "Bye!";
    int sent;
    int res;

    do {
        res = recv(fd, recvbuf, recvbuflen, 0);
        if (res == 0) {
            printf("Peer closed\n");
            break;
        } else if (res == SOCKET_ERROR) {
            sockerr("recv()");
            return;
        }

        /* No error, echo */
        printf("Bytes received: %d\n", res);
        sent = send(fd, recvbuf, res, 0);
        if (sent == SOCKET_ERROR) {
            sockerr("send()");
            return;
        }
        printf("Bytes sent: %d\n", sent);

    } while (res > 0);

    /* Send bye */
    sent = send(fd, byebuf, sizeof(byebuf), 0);
    if (sent == SOCKET_ERROR) {
        sockerr("send() bye");
        return;
    }
    printf("Bye Bytes sent: %d\n", sent);
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


/* The client sends a messages, and waits for the echo before shutting
 * down the send side. It then expects a bye message from the server.
 */
static int client(GUID target)
{
    SOCKET fd = INVALID_SOCKET;
    SOCKADDR_HV sa;
    char *sendbuf = "this is a test";
    char recvbuf[MY_BUFLEN];
    int recvbuflen = MY_BUFLEN;
    int res;

    fd = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
    if (fd == INVALID_SOCKET) {
        sockerr("socket()");
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

    res = send(fd, sendbuf, (int)strlen(sendbuf), 0);
    if (res == SOCKET_ERROR) {
        sockerr("send()");
        goto out;
    }

    printf("Bytes Sent: %d\n", res);

    res = recv(fd, recvbuf, recvbuflen, 0);
    if (res < 0) {
        sockerr("recv()");
        goto out;
    } else if (res == 0) {
        printf("Connection closed\n");
        res = 1;
        goto out;
    }

    printf("Bytes received: %d\n", res);
    printf("->%s\n", recvbuf);
    printf("Shutdown\n");

    /* XXX shutdown does not work! */
    res = shutdown(fd, SD_SEND);
    if (res == SOCKET_ERROR) {
        sockerr("shutdown()");
        goto out;
    }

    printf("Wait for bye\n");
    res = recv(fd, recvbuf, recvbuflen, 0);
    if (res < 0) {
        sockerr("recv()");
        goto out;
    } else if (res == 0) {
        printf("Connection closed\n");
        res = 1;
        goto out;
    }

    printf("Bytes received: %d\n", res);
    printf("->%s\n", recvbuf);
    res = 0;

 out:
    closesocket(fd);
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
        res = client(target);
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
