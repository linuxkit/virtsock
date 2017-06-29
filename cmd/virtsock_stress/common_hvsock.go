package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/linuxkit/virtsock/pkg/hvsock"
)

var (
	svcid, _ = hvsock.GUIDFromString("3049197C-FACB-11E6-BD58-64006A7986D3")
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
	return fmt.Sprintf("%s:%s", cl.vmid.String(), svcid.String())
}

func (cl hvsockClient) Dial(conid int) (Conn, error) {
	sa := hvsock.HypervAddr{VMID: cl.vmid, ServiceID: svcid}
	return hvsock.Dial(sa)
}

func HvsockServerListen() net.Listener {
	fmt.Printf("Listen on port: %s\n", svcid.String())
	l, err := hvsock.Listen(hvsock.HypervAddr{VMID: hvsock.GUIDWildcard, ServiceID: svcid})
	if err != nil {
		log.Fatalln("Listen():", err)
	}

	return l
}
