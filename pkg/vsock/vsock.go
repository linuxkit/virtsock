// Package vsock implements Go bindings to the virtio socket device.
// It primarily provides the bindings for the Linux guest but also
// adds support for the hyperkit based implementation on macOS hosts.
package vsock

import (
	"net"
)

const (
	// CIDAny is a wildcard CID
	CIDAny = 4294967295 // 2^32-1
	// CIDHypervisor is the reserved CID for the Hypervisor
	CIDHypervisor = 0
	// CIDHost is the reserved CID for the host system
	CIDHost = 2
)

// Conn is a vsock connection which supports half-close.
type Conn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}
