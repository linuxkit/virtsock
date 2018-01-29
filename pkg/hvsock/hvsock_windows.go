package hvsock

import (
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

// Make sure Winsock2 is initialised
func init() {
	e := syscall.WSAStartup(uint32(0x202), &wsaData)
	if e != nil {
		log.Fatal("WSAStartup", e)
	}
}

const (
	hvsockAF  = 34 // AF_HYPERV
	hvsockRaw = 1  // SHV_PROTO_RAW
)

var (
	// ErrTimeout is an error returned on timeout
	ErrTimeout = &timeoutError{}

	wsaData syscall.WSAData
)

// Dial a Hyper-V socket address
func Dial(raddr HypervAddr) (Conn, error) {
	fd, err := syscall.Socket(hvsockAF, syscall.SOCK_STREAM, hvsockRaw)
	if err != nil {
		return nil, err
	}

	var sa rawSockaddrHyperv
	ptr, n, err := raddr.sockaddr(&sa)
	if err != nil {
		return nil, err
	}

	if err := sys_connect(fd, ptr, n); err != nil {
		return nil, errors.Wrapf(err, "connect(%s) failed", raddr)
	}

	v, err := newHVsockConn(fd, HypervAddr{VMID: GUIDZero, ServiceID: GUIDZero}, raddr)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Listen returns a net.Listener which can accept connections on the given port
func Listen(addr HypervAddr) (net.Listener, error) {
	return nil, fmt.Errorf("Listen() unimplemented")
}

//
// Hyper-V socket connection implementation
//

// HVsockConn represent a Hyper-V connection. Complex mostly due to asynch send()/recv() syscalls.
type HVsockConn struct {
	fd     syscall.Handle
	local  HypervAddr
	remote HypervAddr

	wg            sync.WaitGroup
	closing       bool
	readDeadline  deadlineHandler
	writeDeadline deadlineHandler
}

func newHVsockConn(h syscall.Handle, local HypervAddr, remote HypervAddr) (*HVsockConn, error) {
	ioInitOnce.Do(initIo)
	v := &HVsockConn{fd: h, local: local, remote: remote}

	_, err := createIoCompletionPort(h, ioCompletionPort, 0, 0xffffffff)
	if err != nil {
		return nil, err
	}
	err = setFileCompletionNotificationModes(h,
		cFILE_SKIP_COMPLETION_PORT_ON_SUCCESS|cFILE_SKIP_SET_EVENT_ON_HANDLE)
	if err != nil {
		return nil, err
	}
	v.readDeadline.channel = make(timeoutChan)
	v.writeDeadline.channel = make(timeoutChan)

	return v, nil
}

// LocalAddr returns the local address of a connection
func (v *HVsockConn) LocalAddr() net.Addr {
	return v.local
}

// RemoteAddr returns the remote address of a connection
func (v *HVsockConn) RemoteAddr() net.Addr {
	return v.remote
}

// Close closes the connection
func (v *HVsockConn) Close() error {
	if !v.closing {
		// cancel all IO and wait for it to complete
		v.closing = true
		cancelIoEx(v.fd, nil)
		v.wg.Wait()
		// at this point, no new IO can start
		syscall.Close(v.fd)
		v.fd = 0
	}
	return nil
}

// CloseRead shuts down the reading side of a hvsock connection
func (v *HVsockConn) CloseRead() error {
	return syscall.Shutdown(v.fd, syscall.SHUT_RD)
}

// CloseWrite shuts down the writing side of a hvsock connection
func (v *HVsockConn) CloseWrite() error {
	return syscall.Shutdown(v.fd, syscall.SHUT_WR)
}

// Read reads data from the connection
func (v *HVsockConn) Read(buf []byte) (int, error) {
	var b syscall.WSABuf
	var f uint32

	b.Len = uint32(len(buf))
	b.Buf = &buf[0]

	c, err := v.prepareIo()
	if err != nil {
		return 0, err
	}

	if v.readDeadline.timedout.isSet() {
		return 0, ErrTimeout
	}

	var bytes uint32
	err = syscall.WSARecv(v.fd, &b, 1, &bytes, &f, &c.o, nil)
	n, err := v.asyncIo(c, &v.readDeadline, bytes, err)
	runtime.KeepAlive(buf)

	// Handle EOF conditions.
	if err == nil && n == 0 && len(buf) != 0 {
		return 0, io.EOF
	} else if err == syscall.ERROR_BROKEN_PIPE {
		return 0, io.EOF
	} else {
		return n, err
	}
}

// Write writes data over the connection
func (v *HVsockConn) Write(buf []byte) (int, error) {
	var b syscall.WSABuf
	var f uint32

	if len(buf) == 0 {
		return 0, nil
	}

	f = 0
	b.Len = uint32(len(buf))
	b.Buf = &buf[0]

	c, err := v.prepareIo()
	if err != nil {
		return 0, err
	}
	if v.writeDeadline.timedout.isSet() {
		return 0, ErrTimeout
	}

	var bytes uint32
	err = syscall.WSASend(v.fd, &b, 1, &bytes, f, &c.o, nil)
	n, err := v.asyncIo(c, &v.writeDeadline, bytes, err)
	runtime.KeepAlive(buf)
	return n, err
}

// SetReadDeadline implementation for Hyper-V sockets
func (v *HVsockConn) SetReadDeadline(deadline time.Time) error {
	return v.readDeadline.set(deadline)
}

// SetWriteDeadline implementation for Hyper-V sockets
func (v *HVsockConn) SetWriteDeadline(deadline time.Time) error {
	return v.writeDeadline.set(deadline)
}

// SetDeadline implementation for Hyper-V sockets
func (v *HVsockConn) SetDeadline(deadline time.Time) error {
	if err := v.SetReadDeadline(deadline); err != nil {
		return err
	}
	return v.SetWriteDeadline(deadline)
}

// Helper functions for conversion to sockaddr

// struck sockaddr equivalent
type rawSockaddrHyperv struct {
	Family    uint16
	Reserved  uint16
	VMID      GUID
	ServiceID GUID
}

// Utility function to build a struct sockaddr for syscalls.
func (a HypervAddr) sockaddr(sa *rawSockaddrHyperv) (unsafe.Pointer, int32, error) {
	sa.Family = hvsockAF
	sa.Reserved = 0
	for i := 0; i < len(sa.VMID); i++ {
		sa.VMID[i] = a.VMID[i]
	}
	for i := 0; i < len(sa.ServiceID); i++ {
		sa.ServiceID[i] = a.ServiceID[i]
	}

	return unsafe.Pointer(sa), int32(unsafe.Sizeof(*sa)), nil
}

// Help for read/write timeouts
type deadlineHandler struct {
	setLock     sync.Mutex
	channel     timeoutChan
	channelLock sync.RWMutex
	timer       *time.Timer
	timedout    atomicBool
}

// The code below here is adjusted from:
// https://github.com/Microsoft/go-winio/blob/master/file.go
type atomicBool int32

func (b *atomicBool) isSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setFalse()   { atomic.StoreInt32((*int32)(b), 0) }
func (b *atomicBool) setTrue()    { atomic.StoreInt32((*int32)(b), 1) }

const (
	cFILE_SKIP_COMPLETION_PORT_ON_SUCCESS = 1
	cFILE_SKIP_SET_EVENT_ON_HANDLE        = 2
)

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

type timeoutChan chan struct{}

var ioInitOnce sync.Once
var ioCompletionPort syscall.Handle

// ioResult contains the result of an asynchronous IO operation
type ioResult struct {
	bytes uint32
	err   error
}

type ioOperation struct {
	o  syscall.Overlapped
	ch chan ioResult
}

func initIo() {
	h, err := createIoCompletionPort(syscall.InvalidHandle, 0, 0, 0xffffffff)
	if err != nil {
		panic(err)
	}
	ioCompletionPort = h
	go ioCompletionProcessor(h)
}

// prepareIo prepares for a new IO operation
func (v *HVsockConn) prepareIo() (*ioOperation, error) {
	v.wg.Add(1)
	if v.closing {
		return nil, fmt.Errorf("HvSocket has already been closed")
	}
	c := &ioOperation{}
	c.ch = make(chan ioResult)
	return c, nil
}

// ioCompletionProcessor processes completed async IOs forever
func ioCompletionProcessor(h syscall.Handle) {
	// Set the timer resolution to 1. This fixes a performance regression in golang 1.6.
	timeBeginPeriod(1)
	for {
		var bytes uint32
		var key uintptr
		var op *ioOperation
		err := getQueuedCompletionStatus(h, &bytes, &key, &op, syscall.INFINITE)
		if op == nil {
			panic(err)
		}
		op.ch <- ioResult{bytes, err}
	}
}

// asyncIo processes the return value from Recv or Send, blocking until
// the operation has actually completed.
func (v *HVsockConn) asyncIo(c *ioOperation, d *deadlineHandler, bytes uint32, err error) (int, error) {
	if err != syscall.ERROR_IO_PENDING {
		v.wg.Done()
		return int(bytes), err
	}

	if v.closing {
		cancelIoEx(v.fd, &c.o)
	}

	var timeout timeoutChan
	if d != nil {
		d.channelLock.Lock()
		timeout = d.channel
		d.channelLock.Unlock()
	}

	var r ioResult
	select {
	case r = <-c.ch:
		err = r.err
		if err == syscall.ERROR_OPERATION_ABORTED {
			if v.closing {
				err = fmt.Errorf("HvSocket has already been closed")
			}
		}
	case <-timeout:
		cancelIoEx(v.fd, &c.o)
		r = <-c.ch
		err = r.err
		if err == syscall.ERROR_OPERATION_ABORTED {
			err = ErrTimeout
		}
	}

	// runtime.KeepAlive is needed, as c is passed via native
	// code to ioCompletionProcessor, c must remain alive
	// until the channel read is complete.
	runtime.KeepAlive(c)
	v.wg.Done()
	return int(r.bytes), err
}

func (d *deadlineHandler) set(deadline time.Time) error {
	d.setLock.Lock()
	defer d.setLock.Unlock()

	if d.timer != nil {
		if !d.timer.Stop() {
			<-d.channel
		}
		d.timer = nil
	}
	d.timedout.setFalse()

	select {
	case <-d.channel:
		d.channelLock.Lock()
		d.channel = make(chan struct{})
		d.channelLock.Unlock()
	default:
	}

	if deadline.IsZero() {
		return nil
	}

	timeoutIO := func() {
		d.timedout.setTrue()
		close(d.channel)
	}

	now := time.Now()
	duration := deadline.Sub(now)
	if deadline.After(now) {
		// Deadline is in the future, set a timer to wait
		d.timer = time.AfterFunc(duration, timeoutIO)
	} else {
		// Deadline is in the past. Cancel all pending IO now.
		timeoutIO()
	}
	return nil
}
