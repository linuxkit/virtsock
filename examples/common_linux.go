package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/linuxkit/virtsock/pkg/vsock"
)

var (
	socketMode string
)

func init() {
	flag.StringVar(&socketMode, "m", "", "Socket Mode (vsock or hvsock)")
}

func SetVerbosity() {
	HvsockSetVerbosity()
	//VsockSetVerbosity() // No verbosity settings there, yet
}

func ValidateOptions() {
	if socketMode == "" {
		if _, err := os.Stat("/sys/bus/vmbus"); err != nil && os.IsNotExist(err) {
			socketMode = "vsock"
		} else {
			socketMode = "hvsock"
		}
	}
	if socketMode != "hvsock" && socketMode != "vsock" {
		log.Fatalln("Unknown socket mode: ", socketMode)
	}
}

type vsockClient struct {
	cid uint
}

func ParseClientStr(clientStr string) Client {
	if socketMode == "hvsock" {
		return HvsockParseClientStr(clientStr)
	} else if socketMode == "vsock" {
		cid := VsockParseClientStr(clientStr)
		return &vsockClient{cid}
	}
	panic("socketMode")
}

func (cl vsockClient) String() string {
	return fmt.Sprintf("%08x.%08x", cl.cid, vsockPort)
}

func (cl vsockClient) Dial(conid int) (Conn, error) {
	return vsock.Dial(cl.cid, vsockPort)
}

func ServerListen() net.Listener {
	if socketMode == "hvsock" {
		return HvsockServerListen()
	} else if socketMode != "vsock" {
		panic("socketMode")
	}

	l, err := vsock.Listen(vsock.CIDAny, vsockPort)
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	return l
}
