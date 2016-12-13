_Note:_ `virtsock_echo` can be substituted for `virtsock_stress` mostly as is
(the basic options are common to both).

# Operating System Specific

## Windows

Under Windows, we assume Docker for Windows is installed since this ships with a suitably patched Linux kernel (though the tests can also be run to a Windows VM).  Currently, there is also the restriction that one can only connect from the host to the (Linux) VM.

Build a Linux container with the test program as shown below.

## Linux

After building the binary can be run from a suitably privileged
container:

    $ cat >Dockerfile <<EOF
    FROM alpine
    RUN apk update && apk add strace
    ADD virtsock_stress.linux /virtsock_stress
    ENTRYPOINT ["/virtsock_stress"]
    EOF
    $ docker build -t stress . && docker run -it --rm --net=host --privileged stress [...options...]

## MacOS

Under MacOS the default is to assume Hyperkit as configured by Docker
for Mac (since the path to the sockets and the names of the sockets
themselves differ).

To run against standalone Hyperkit the path to the sockets must be
specified when starting Hyperkit and must be passed to the option:

    macos$ ./virtsock_stress.darwin -s -m hyperkit:/var/run/

(this assumes hyperkit was built without `PRI_ADDR_PREFIX` or
`CONNECT_SOCKET_NAME` set at build time and run with e.g. `-s
7,virtio-sock,guest_cid=3,path=/var/run`)

In Docker mode everything is implied to be as it is configured by
Docker for Mac. This is the default but can be given explicitly with:

    macos$ ./virtsock_stress.darwin -s -m docker

# Specific OS Pairs

## Linux & Docker for Windows

Start the linux container with program in server mode:

    PS> docker run -it --rm --net=host --privileged stress -s

The start the client in a separate powershell window:

    PS> $vmId = (Get-VM MobyLinuxVM).Id
    PS> .\virtsock_stress.exe -c $vmId
    

## Linux & Docker for Mac

When running as a client on the Linux side the correct address is cid
"2" (the host):

    linux$ docker run -it --rm --net=host --privileged stress -c 2
    macos$ ./virtsock_stress.darwin -s

When running as a client on the MacOS side the correct address is cid
is "3" (the guest, as configued by Docker for Mac):

    linux$ docker run -it --rm --net=host --privileged stress -s
    macos$ ./virtsock_stress.darwin -c 3
