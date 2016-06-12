/*
 * A Hyper-V socket benchmarking program
 */
#include "compat.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* 3049197C-9A4E-4FBF-9367-97F792F16994 */
DEFINE_GUID(BM_GUID,
    0x3049197c, 0x9a4e, 0x4fbf, 0x93, 0x67, 0x97, 0xf7, 0x92, 0xf1, 0x69, 0x94);

#ifdef _MSC_VER
static WSADATA wsaData;
#endif

#ifndef ARRAY_SIZE
#define ARRAY_SIZE(_arr) (sizeof(_arr)/sizeof(*(_arr)))
#endif

/* Use a static buffer for send and receive. */
#define MAX_BUF_LEN (2 * 1024 * 1024)
static char buf[MAX_BUF_LEN];

/* Amount of data to send per bandwidth iteration */
#define HV_BM_BW_DATA ((uint64_t)1024 * 1024 * 1024)

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

/* Bandwidth tests:
 *
 * The TX side sends a fixed amount of data in fixed sized
 * messages. The RX side drains the ring in message sized chunks (or less).
 */
static int bw_rx(SOCKET fd, int msg_sz)
{
    int ret;
    int rx_sz;

    rx_sz = msg_sz ? msg_sz : ARRAY_SIZE(buf);

    DBG("bw_rx: msg_sz=%d rx_sz=%d\n", msg_sz, rx_sz);

    for (;;) {
        ret = recv(fd, buf, rx_sz, 0);
        if (ret == 0) {
            break;
        } else if (ret == SOCKET_ERROR) {
            sockerr("recv()");
            ret = -1;
            goto err_out;
        }
        TRC("Received: %d\n", ret);
    }
    ret = 0;

err_out:
    return ret;
}

static int bw_tx(SOCKET fd, int msg_sz, uint64_t *bw)
{
    uint64_t total_sent = 0;
    uint64_t start, end, diff;
    int tx_sz;
    int sent;
    int ret;

    tx_sz = msg_sz ? msg_sz : ARRAY_SIZE(buf);

    DBG("bw_tx: msg_sz=%d tx_sz=%d \n", msg_sz, tx_sz);

    start = time_ns();

    while (total_sent < HV_BM_BW_DATA) {
        sent = 0;
        while (sent < tx_sz) {
            ret = send(fd, buf + sent, tx_sz - sent, 0);
            if (ret == SOCKET_ERROR) {
                sockerr("send()");
                ret = -1;
                goto err_out;
            }
            TRC("Sent: %d %d\n", sent, ret);
            sent += ret;
        }
        total_sent += sent;
    }

    end = time_ns();
    diff = end - start;

    /* Bandwidth in Mbits per second */
    *bw = (8 * HV_BM_BW_DATA * 1000000000) / (diff * 1024 * 1024);
    ret = 0;

err_out:
    return ret;
}


/*
 * Main server and client entry points
 */
static int server(int bw, int msg_sz)
{
    SOCKET lsock, csock;
    SOCKADDR_HV sa, sac;
    socklen_t socklen = sizeof(sac);
    int ret = 0;

    INFO("server: bw=%d msg_sz=%d\n", bw, msg_sz);

    lsock = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
    if (lsock == INVALID_SOCKET) {
        sockerr("socket()");
        return -1;
    }

    memset(&sa, 0, sizeof(sa));
    sa.Family = AF_HYPERV;
    sa.Reserved = 0;
    sa.VmId = HV_GUID_WILDCARD;
    sa.ServiceId = BM_GUID;

    ret = bind(lsock, (const struct sockaddr *)&sa, sizeof(sa));
    if (ret == SOCKET_ERROR) {
        sockerr("bind()");
        closesocket(lsock);
        return -1;
    }

    INFO("server: listening\n");

    ret = listen(lsock, SOMAXCONN);
    if (ret == SOCKET_ERROR) {
        sockerr("listen()");
        closesocket(lsock);
        return -1;
    }

    memset(&sac, 0, sizeof(sac));
    csock = accept(lsock, (struct sockaddr *)&sac, &socklen);
    if (csock == INVALID_SOCKET) {
        sockerr("accept()");
        closesocket(lsock);
        return -1;
    }

    INFO("server: accepted\n");

    if (bw)
        ret = bw_rx(csock, msg_sz);

    closesocket(csock);
    closesocket(lsock);
    return ret;
}


static int client(GUID target, int bw, int msg_sz)
{
    SOCKET fd;
    SOCKADDR_HV sa;
    uint64_t res;
    int ret = 0;

    INFO("client: bw=%d msg_sz=%d\n", bw, msg_sz);

    fd = socket(AF_HYPERV, SOCK_STREAM, HV_PROTOCOL_RAW);
    if (fd == INVALID_SOCKET) {
        sockerr("socket()");
        return -1;
    }

    memset(&sa, 0, sizeof(sa));
    sa.Family = AF_HYPERV;
    sa.Reserved = 0;
    sa.VmId = target;
    sa.ServiceId = BM_GUID;

    ret = connect(fd, (const struct sockaddr *)&sa, sizeof(sa));
    if (ret == SOCKET_ERROR) {
        sockerr("connect()");
        ret = -1;
        goto err_out;
    }

    INFO("client: connected\n");

    if (bw) {
        ret = bw_tx(fd, msg_sz, &res);
        if (ret)
            goto err_out;
        printf("%d %"PRIu64"\n", msg_sz, res);
    }

err_out:
    closesocket(fd);
    return ret;
}

void usage(char *name)
{
    printf("%s: -s|-c <carg> -b|-l -m <sz> [-v]\n", name);
    printf(" -s        Server mode\n");
    printf(" -c <carg> Client mode. <carg>:\n");
    printf("   'loopback': Connect in loopback mode\n");
    printf("   'parent':   Connect to the parent partition\n");
    printf("   <guid>:     Connect to VM with GUID\n");
    printf(" -b        Bandwidth test\n");
    printf(" -l        Latency test\n");
    printf(" -m <sz>   Message size in bytes\n");
    printf(" -v        Verbose output\n");
}

int __cdecl main(int argc, char **argv)
{
    int opt_server = 0;
    int opt_bw = 0;
    int opt_msgsz = 0;
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

        } else if (strcmp(argv[i], "-b") == 0) {
            opt_bw = 1;
        } else if (strcmp(argv[i], "-l") == 0) {
            opt_bw = 0;
        } else if (strcmp(argv[i], "-m") == 0) {
            if (i + 1 >= argc) {
                fprintf(stderr, "-m requires an argument\n");
                usage(argv[0]);
                goto out;
            }
            opt_msgsz = atoi(argv[++i]);
        } else if (strcmp(argv[i], "-v") == 0) {
            verbose++;
        } else {
            usage(argv[0]);
            goto out;
        }
    }

    if (!opt_bw) {
        fprintf(stderr, "Latency tests currently not implemented\n");
        goto out;
    }

    if (opt_server)
        res = server(opt_bw, opt_msgsz);
    else
        res = client(target, opt_bw, opt_msgsz);

out:
#ifdef _MSC_VER
    WSACleanup();
#endif
    return res;
}
