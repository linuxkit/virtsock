package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var (
	socketMode string

	socketPath  string
	connectPath string
	socketFmt   string
)

func init() {
	flag.StringVar(&socketMode, "m", "docker", "Socket Mode (hyperkit:/path/ or docker)")
}

func SetVerbosity() {}

func ValidateOptions() {
	if strings.HasPrefix(socketMode, "hyperkit:") {
		socketPath = socketMode[len("hyperkit:"):]
		connectPath = filepath.Join(socketPath, "connect")
		socketFmt = "%08x.%08x"
	} else if socketMode == "docker" {
		socketPath = filepath.Join(os.Getenv("HOME"), "/Library/Containers/com.docker.docker/Data")
		connectPath = filepath.Join(socketPath, "@connect")
		socketFmt = "*%08x.%08x"
	} else {
		log.Fatalln("Unknown socket mode: ", socketMode)
	}
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
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{connectPath, "unix"})
	if err != nil {
		return c, err
	}
	if _, err := fmt.Fprintf(c, "%s\n", cl.String()); err != nil {
		return c, fmt.Errorf("Failed to write dest (%s) to %s", cl, connectPath)
	}
	return c, nil
}

func ServerListen() net.Listener {
	sock := filepath.Join(socketPath, fmt.Sprintf(socketFmt, 2, vsockPort))
	if err := os.Remove(sock); err != nil && !os.IsNotExist(err) {
		log.Fatalln("Listen(): Remove:", err)
	}

	l, err := net.ListenUnix("unix", &net.UnixAddr{sock, "unix"})
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	return l
}
