package main

import (
	"log"
	"net"
	"strings"

	"github.com/linuxkit/virtsock/pkg/hvsock"
)

var (
	svcid, _ = hvsock.GUIDFromString("3049197C-FACB-11E6-BD58-64006A7986D3")
)

type hvsockAddr struct {
	addr hvsock.HypervAddr
}

// hvsockParseSockStr extracts the vmid and svcid from a string.
// The format is "VMID:Service", "VMID", or ":Service" as well as an
// empty string. For VMID we also support "parent" and assume
// "loopback" if the string can't be parsed.
func hvsockParseSockStr(sockStr string) hvsockAddr {
	a := hvsock.HypervAddr{hvsock.GUIDZero, svcid}
	if sockStr == "" {
		return hvsockAddr{a}
	}

	var err error
	vmStr := ""
	svcStr := ""
	if strings.Contains(sockStr, ":") {
		vmStr, svcStr, err = net.SplitHostPort(sockStr)
		if err != nil {
			log.Fatalf("Error parsing socket string '%s': %v", sockStr, err)
		}
	} else {
		vmStr = sockStr
	}

	if vmStr != "" {
		if strings.Contains(vmStr, "-") {
			a.VMID, err = hvsock.GUIDFromString(vmStr)
			if err != nil {
				log.Fatalf("Error parsing VM '%s': %v", vmStr, err)
			}
		} else if clientStr == "parent" {
			a.VMID = hvsock.GUIDParent
		} else {
			a.VMID = hvsock.GUIDLoopback
		}
	}

	if svcStr != "" {
		a.ServiceID, err = hvsock.GUIDFromString(svcStr)
		if err != nil {
			log.Fatalf("Error parsing SVC '%s': %v", svcStr, err)
		}
	}
	return hvsockAddr{a}
}

func (s hvsockAddr) String() string {
	return s.addr.String()
}

// Dial connects on a Hyper-V socket
func (s hvsockAddr) Dial(conid int) (Conn, error) {
	return hvsock.Dial(s.addr)
}

// Listen returns a net.Listener for a given Hyper-V socket
func (s hvsockAddr) Listen() net.Listener {
	l, err := hvsock.Listen(s.addr)
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	return l
}

// ListenPacket is not implemented for Hyper-V sockets
func (s hvsockAddr) ListenPacket() net.PacketConn {
	log.Fatalln("ListenPacket(): not implemented for Hyper-V sockets")
	return nil
}
