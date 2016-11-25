package main

import (
	"log"
	"net"
	"strings"

	"../hvsock"
)

var (
	svcid, _ = hvsock.GuidFromString("3049197C-9A4E-4FBF-9367-97F792F16994")
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
	vmid := hvsock.GUID_ZERO
	var err error
	if strings.Contains(clientStr, "-") {
		vmid, err = hvsock.GuidFromString(clientStr)
		if err != nil {
			log.Fatalln("Can't parse GUID: ", clientStr)
		}
	} else if clientStr == "parent" {
		vmid = hvsock.GUID_PARENT
	} else {
		vmid = hvsock.GUID_LOOPBACK
	}

	return hvsockClient{vmid}
}

func (cl hvsockClient) String() string {
	return cl.vmid.String()
}

func (cl hvsockClient) Dial(conid int) (Conn, error) {
	sa := hvsock.HypervAddr{VmId: cl.vmid, ServiceId: svcid}
	return hvsock.Dial(sa)
}

func HvsockServerListen() net.Listener {
	l, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		log.Fatalln("Listen():", err)
	}

	return l
}
