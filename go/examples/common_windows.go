package main

import (
	"net"
)

func SetVerbosity() {
	HvsockSetVerbosity()
}

func ValidateOptions() {}

func ParseClientStr(clientStr string) Client {
	return HvsockParseClientStr(clientStr)
}

func ServerListen() net.Listener {
	return HvsockServerListen()
}
