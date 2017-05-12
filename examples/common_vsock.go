package main

import (
	"log"
	"strconv"
)

const (
	vsockPort = 0x5653
)

func VsockParseClientStr(clientStr string) uint32 {
	cid, err := strconv.ParseUint(clientStr, 10, 32)
	if err != nil {
		log.Fatalf("Can't convert %s to a uint32.", clientStr, err)
	}
	return uint32(cid)
}
