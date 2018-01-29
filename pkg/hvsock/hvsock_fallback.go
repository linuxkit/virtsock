// +build !linux,!windows

package hvsock

import (
	"fmt"
	"net"
	"runtime"
)

func Dial(raddr HypervAddr) (Conn, error) {
	return nil, fmt.Errorf("Dial() not implemented on %s", runtime.GOOS)
}

func Listen(addr HypervAddr) (net.Listener, error) {
	return nil, fmt.Errorf("Listen() not implemented on %s", runtime.GOOS)
}
