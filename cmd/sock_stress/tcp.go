package main

import (
	"log"
	"net"
	"strings"
)

const (
	tcpPort = "5303"
)

type tcpAddr struct {
	addr *net.TCPAddr
	net  string
}

// tcpParseSockStr extracts the host and port from a string.
// The format is "Host:Port", "Host", or ":Port" as well as an empty string.
func tcpParseSockStr(netStr, sockStr string) tcpAddr {
	var s tcpAddr
	s.net = netStr

	if netStr == "tcp6" {
		// IPv6 address. Append port if not specified
		if strings.Contains(sockStr, "]") {
			if sockStr[len(sockStr)-1:] == "]" {
				// IPv6 address bu no port
				sockStr = sockStr + ":" + tcpPort
			}
		} else {
			if !strings.Contains(sockStr, ":") {
				// empty host portion or hostname given
				// but no port. Append port
				sockStr = sockStr + ":" + tcpPort
			}
		}
	} else {
		// IPv4 address. Append port if not specified
		if !strings.Contains(sockStr, ":") {
			sockStr = sockStr + ":" + tcpPort
		}
	}

	a, err := net.ResolveTCPAddr(netStr, sockStr)
	if err != nil {
		log.Fatalf("Error parsing socket string '%s': %v", sockStr, err)
	}
	s.addr = a
	return s
}

func (s tcpAddr) String() string {
	return s.net + "://" + s.addr.String()
}

// Dial connects on a TCP socket
func (s tcpAddr) Dial(conid int) (Conn, error) {
	c, err := net.Dial(s.net, s.addr.String())
	if err != nil {
		return nil, err
	}
	return c.(*net.TCPConn), err
}

// Listen returns a net.Listener for a given TCP socket
func (s tcpAddr) Listen() net.Listener {
	l, err := net.Listen(s.net, s.addr.String())
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	return l
}
