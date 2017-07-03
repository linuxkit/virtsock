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
	net  string
	host string
	port string
}

// tcpParseSockStr extracts the host and port from a string.
// The format is "Host:Port", "Host", or ":Port" as well as an empty string.
func tcpParseSockStr(netStr, sockStr string) tcpAddr {
	a := tcpAddr{net: netStr, host: "localhost", port: tcpPort}
	if sockStr == "" {
		return a
	}

	var err error
	if strings.Contains(sockStr, ":") {
		a.host, a.port, err = net.SplitHostPort(sockStr)
		if err != nil {
			log.Fatalf("Error parsing socket string '%s': %v", sockStr, err)
		}
	} else {
		a.host = sockStr
	}
	return a
}

func (s tcpAddr) String() string {
	return s.net + "://" + s.host + ":" + s.port
}

// Dial connects on a virtio socket
func (s tcpAddr) Dial(conid int) (Conn, error) {
	c, err := net.Dial(s.net, s.host+":"+s.port)
	if err != nil {
		return nil, err
	}
	return c.(*net.TCPConn), err
}

// Listen returns a net.Listener for a given virtio socket
func (s tcpAddr) Listen() net.Listener {
	l, err := net.Listen(s.net, s.host+":"+s.port)
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	return l
}
