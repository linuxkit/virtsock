package main

import (
	"log"
	"net"
	"strings"

	"github.com/rneugeba/virtsock/pkg/hvsock"
)

var (
	svcid, _ = hvsock.GUIDFromString("3049197C-9A4E-4FBF-9367-97F792F16994")
)

func HvsockSetVerbosity() {
	if verbose > 2 {
		hvsock.Debug = true
	}
}

type hvsockClient struct {
	vmid hvsock.GUID
}

func HvsockParseClientStr(clientStr string) hvsockClient {
	vmid := hvsock.GUIDZero
	var err error
	if strings.Contains(clientStr, "-") {
		vmid, err = hvsock.GUIDFromString(clientStr)
		if err != nil {
			log.Fatalln("Can't parse GUID: ", clientStr)
		}
	} else if clientStr == "parent" {
		vmid = hvsock.GUIDParent
	} else {
		vmid = hvsock.GUIDLoopback
	}

	return hvsockClient{vmid}
}

func (cl hvsockClient) String() string {
	return cl.vmid.String()
}

func (cl hvsockClient) Dial(conid int) (Conn, error) {
	sa := hvsock.HypervAddr{VMID: cl.vmid, ServiceID: svcid}
	return hvsock.Dial(sa)
}

func HvsockServerListen() net.Listener {
	l, err := hvsock.Listen(hvsock.HypervAddr{VMID: hvsock.GUIDWildcard, ServiceID: svcid})
	if err != nil {
		log.Fatalln("Listen():", err)
	}

	return l
}
