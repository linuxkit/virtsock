_Note:_ `hvgoecho` can be substituted for `hvgostress` mostly as is
(the basic options are common to both).

# Operating System Specific

## Windows

TBD

## Linux

After building the binary can be run from a suitably privileged
container:

    $ cat >Dockerfile <<EOF
    FROM alpine
    RUN apk update && apk add strace
    ADD hvgostress.linux /hvgostress
    ENTRYPOINT ["/hvgostress"]
    EOF
    $ docker build -t stress . && docker run -it --rm --net=host --privileged stress [...options...]

## MacOS

Under MacOS the default is to assume Hyperkit as configured by Docker
for Mac (since the path to the sockets and the names of the sockets
themselves differ).

To run against standalone Hyperkit the path to the sockets must be
specified when starting Hyperkit and must be passed to the option:

    macos$ ./hvgostress.darwin -s -m hyperkit:/var/run/

(this assumes hyperkit was built without `PRI_ADDR_PREFIX` or
`CONNECT_SOCKET_NAME` set at build time and run with e.g. `-s
7,virtio-sock,guest_cid=3,path=/var/run`)

In Docker mode everything is implied to be as it is configured by
Docker for Mac. This is the default but can be given explicitly with:

    macos$ ./hvgostress.darwin -s -m docker

# Specific OS Pairs

## Linux & Docker for Windows

TBD

## Linux & Docker for Mac

When running as a client on the Linux side the correct address is cid
"2" (the host):

    linux$ docker run -it --rm --net=host --privileged stress -c 2
    macos$ ./hvgostress.darwin -s

When running as a client on the MacOS side the correct address is cid
is "3" (the guest, as configued by Docker for Mac):

    linux$ docker run -it --rm --net=host --privileged stress -s
    macos$ ./hvgostress.darwin -c 3
