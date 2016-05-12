# Hyper-V Sockets echo application

This is a simple echo application utilising Hyper-V sockets which uses
`shutdown()` for a simple handshake to close the connection.  The
server accepts connections from anywhere and echos back what ever it
receives.  When it receives 0 bytes (indicating that the client closed
the connection) it will attempt to send a "bye" message to the client.
The client, will connect to the server and then send a message and
waits for the reply.  Once it received the reply it will close the
send/write side of the connection and waits to receive the "bye"
message from the server.

The echo application can be compiled and run on Windows and Linux
(assuming the Linux kernel has the Hyper-V socket patches applied).


# Building

To build on Windows either open the solution in Visual Studio or run
`msbuild`. You must have a SDK installed with a minimum version of
14290. The project assumes 14291, so you may have to adjust it.

To build the Linux executable type `make hvecho`. To build for
MobyLinux simply type `make` (this assumes you compile with Docker for
Mac, untested on Windows).


# Running

On the windows host, you have to register the service *once* using:
```
$service = New-Item -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices" -Name 3049197C-9A4E-4FBF-9367-97F792F16994

$service.SetValue("ElementName", "Hyper-V Socket Echo Service")
```

The echo server is started with:
```
hvecho -s
```

The client supports a number of modes. By default, with no arguments supplied it will connected using loopback mode, i.e. it tries to connect to the server on the same partition:
```
hvecho -c
```

The client can be run in a VM and started with:
```
hvecho -c parent
```
which attempts to connect to the server in the parent partition.


Finally, if the server is run in a VM, the client can be invoked in the parent partition with:
```
client -c <vmid>
```
where `<vmid>` is the GUID of the VM the server is running in. The GUID can be retrieved with: `(get-vm <VM Name>).id`.
