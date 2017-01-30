package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/rneugeba/virtsock/pkg/vsock"
)

var (
	socketMode string
)

func init() {
	flag.StringVar(&socketMode, "m", "docker", "Socket Mode (hyperkit:/path/ or docker)")
}

func SetVerbosity() {}

func ValidateOptions() {
	vsock.SocketMode(socketMode)
}

type vsockClient struct {
	cid uint
}

func ParseClientStr(clientStr string) Client {
	cid := VsockParseClientStr(clientStr)
	return &vsockClient{cid}
}

func (cl vsockClient) String() string {
	return fmt.Sprintf("%08x.%08x", cl.cid, vsockPort)
}

func (cl vsockClient) Dial(conid int) (Conn, error) {
	return vsock.Dial(cl.cid, vsockPort)
}

func ServerListen() net.Listener {
	l, err := vsock.Listen(vsock.CIDHost, vsockPort)
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	return l
}
