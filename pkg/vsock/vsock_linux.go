// This provides the Linux guest bindings to the VM sockets. VM
// sockets are a generic mechanism for guest<->host communication. It
// was originally developed for VMware but now also supports virtio
// sockets and (soon) Hyper-V sockets.

package vsock

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"
	"syscall"
	
	"golang.org/x/sys/unix"
)

// SocketMode is a NOOP on Linux
func SocketMode(m string) {
}

// Dial connects to the CID.Port via virtio sockets
func Dial(cid, port uint32) (Conn, error) {
	fd, err := syscall.Socket(unix.AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	sa := &unix.SockaddrVM{CID: cid, Port: port}
	if err = unix.Connect(fd, sa); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"failed connect() to %08x.%08x: %s", cid, port, err))
	}
	return newVsockConn(uintptr(fd), port)
}

// Listen returns a net.Listener which can accept connections on the given port
func Listen(cid, port uint32) (net.Listener, error) {
	fd, err := syscall.Socket(unix.AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	sa := &unix.SockaddrVM{CID: cid, Port: port}
	if err = unix.Bind(fd, sa); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"failed bind() to %08x.%08x: %s", cid, port, err))
	}
	
	err = syscall.Listen(fd, syscall.SOMAXCONN)
	if err != nil {
		return nil, err
	}
	return &vsockListener{fd, cid, port}, nil
}

type vsockListener struct {
	fd   int
	cid  uint32
	port uint32
}

// Accept implements the Accept method in the Listener interface; it
// waits for the next call and returns a generic Conn.
func (v *vsockListener) Accept() (net.Conn, error) {
	fd, _, err := unix.Accept(v.fd)
	if err != nil {
		return nil, err
	}
	return newVsockConn(uintptr(fd), v.port)
}

// Close implements the Close method in the Listener interface; it
// closes the Listening connection.
func (v *vsockListener) Close() error {
	// Note this won't cause the Accept to unblock.
	return unix.Close(v.fd)
}


//
type VsockAddr struct {
	CID uint32
	Port uint32
}

func (a VsockAddr) Network() string {
	return "vsock"
}

func (a VsockAddr) String() string {
	return fmt.Sprintf("%08x.%08x", a.CID, a.Port)
}

func (v *vsockListener) Addr() net.Addr {
	return VsockAddr{CID: v.cid, Port: v.port}
}

// a wrapper around FileConn which supports CloseRead and CloseWrite
type vsockConn struct {
	vsock  *os.File
	fd     uintptr
	local  VsockAddr
	remote VsockAddr
}

type VsockConn struct {
	vsockConn
}

func newVsockConn(fd uintptr, lCID, lPort, rCID, rPort uint32) (*VsockConn, error) {
	vsock := os.NewFile(fd, fmt.Sprintf("vsock:%d", fd))
	local := VsockAddr{CID: lCID, Port: lPort}
	remote := VsockAddr{CID: rCID, Port: rPort}
	return &VsockConn{vsockConn{vsock: vsock, fd: fd, local: local, remote: remote}}, nil
}

func (v *VsockConn) LocalAddr() net.Addr {
	return v.local
}

func (v *VsockConn) RemoteAddr() net.Addr {
	return v.remote
}

func (v *VsockConn) CloseRead() error {
	return syscall.Shutdown(int(v.fd), syscall.SHUT_RD)
}

func (v *VsockConn) CloseWrite() error {
	return syscall.Shutdown(int(v.fd), syscall.SHUT_WR)
}

func (v *VsockConn) Close() error {
	return v.vsock.Close()
}

func (v *VsockConn) Read(buf []byte) (int, error) {
	return v.vsock.Read(buf)
}

func (v *VsockConn) Write(buf []byte) (int, error) {
	return v.vsock.Write(buf)
}

func (v *VsockConn) SetDeadline(t time.Time) error {
	return nil // FIXME
}

func (v *VsockConn) SetReadDeadline(t time.Time) error {
	return nil // FIXME
}

func (v *VsockConn) SetWriteDeadline(t time.Time) error {
	return nil // FIXME
}
