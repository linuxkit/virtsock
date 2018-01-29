// Package hvsock provides a Go interface to Hyper-V sockets both on
// Windows and on Linux. The Linux bindings require patches for the
// 4.9.x kernel. If you are using a Linux kernel 4.14.x or newer you
// should use the vsock package instead as the Hyper-V socket support
// in these kernels have been merged with the virtio sockets
// implementation.
package hvsock

import (
	"fmt"
	"net"
)

var (
	// GUIDZero used by listeners to accept connections from all partitions
	GUIDZero, _ = GUIDFromString("00000000-0000-0000-0000-000000000000")
	// GUIDWildcard used by listeners to accept connections from all partitions
	GUIDWildcard, _ = GUIDFromString("00000000-0000-0000-0000-000000000000")
	// GUIDBroadcast undocumented
	GUIDBroadcast, _ = GUIDFromString("FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF")
	// GUIDChildren used by listeners to accept connections from children
	GUIDChildren, _ = GUIDFromString("90db8b89-0d35-4f79-8ce9-49ea0ac8b7cd")
	// GUIDLoopback use to connect in loopback mode
	GUIDLoopback, _ = GUIDFromString("e0e16197-dd56-4a10-9195-5ee7a155a838")
	// GUIDParent use to connect to the parent partition
	GUIDParent, _ = GUIDFromString("a42e7cda-d03f-480c-9cc2-a4de20abb878")
)

// GUID is used by Hypper-V sockets for "addresses" and "ports"
type GUID [16]byte

// Convert a GUID into a string
func (g *GUID) String() string {
	/* XXX This assume little endian */
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		g[3], g[2], g[1], g[0],
		g[5], g[4],
		g[7], g[6],
		g[8], g[9],
		g[10], g[11], g[12], g[13], g[14], g[15])
}

// GUIDFromString parses a string and returns a GUID
func GUIDFromString(s string) (GUID, error) {
	var g GUID
	var err error
	_, err = fmt.Sscanf(s, "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		&g[3], &g[2], &g[1], &g[0],
		&g[5], &g[4],
		&g[7], &g[6],
		&g[8], &g[9],
		&g[10], &g[11], &g[12], &g[13], &g[14], &g[15])
	return g, err
}

// HypervAddr combined "address" and "port" structure
type HypervAddr struct {
	VMID      GUID
	ServiceID GUID
}

// Network returns the type of network for Hyper-V sockets
func (a HypervAddr) Network() string { return "hvsock" }

func (a HypervAddr) String() string {
	vmid := a.VMID.String()
	svc := a.ServiceID.String()

	return vmid + ":" + svc
}

// Conn is a hvsock connection which supports half-close.
type Conn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}
