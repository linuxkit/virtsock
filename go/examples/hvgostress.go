package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"crypto/md5"
	"math/rand"

	"../"
)

var (
	clientStr   string
	serverMode  bool
	maxDataLen  int
	connections int
	sleepTime   int

	svcid, _ = hvsock.GuidFromString("3049197C-9A4E-4FBF-9367-97F792F16994")
)

func init() {
	flag.StringVar(&clientStr, "c", "", "Client")
	flag.BoolVar(&serverMode, "s", false, "Start as a Server")
	flag.IntVar(&maxDataLen, "l", 64*1024, "Maximum Length of data")
	flag.IntVar(&connections, "i", 100, "Total number of connections")
	flag.IntVar(&sleepTime, "w", 0, "Sleep time in seconds between new connections")

	rand.Seed(time.Now().UnixNano())
}

func randBuf(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func server() {
	l, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	defer func() {
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln("Accept(): ", err)
		}
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		go handleRequest(conn)
	}
}

func handleRequest(c net.Conn) {
	defer func() {
		fmt.Printf("Closing\n")
		err := c.Close()
		if err != nil {
			log.Fatalln("Close():", err)
		}
	}()

	n, err := io.Copy(c, c)
	if err != nil {
		log.Println("Copy():", err)
		return
	}
	fmt.Printf("Copied Bytes: %d\n", n)
	if n == 0 {
		return
	}

	fmt.Printf("Sending BYE message\n")
	// The '\n' is important as the client use ReadString()
	_, err = fmt.Fprintf(c, "Got %d bytes. Bye\n", n)
	if err != nil {
		log.Println("Failed to send: ", err)
		return
	}
	fmt.Printf("Sent bye\n")
}

func client(vmid hvsock.GUID, conid int) {
	sa := hvsock.HypervAddr{VmId: vmid, ServiceId: svcid}
	c, err := hvsock.Dial(sa)
	if err != nil {
		log.Fatalf("[%05d]Failed to Dial: %s:%s %s\n", conid, sa.VmId.String(), sa.ServiceId.String(), err)
	}

	defer c.Close()

	// Create buffer with random data and random length.
	// Make sure the buffer is not zero-length
	buflen := rand.Intn(maxDataLen-1) + 1
	txbuf := randBuf(buflen)
	csum0 := md5.Sum(txbuf)

	fmt.Printf("[%05d] TX: %d bytes, md5=%02x\n", conid, buflen, csum0)

	w := make(chan int)
	go func() {
		l, err := c.Write(txbuf)
		if err != nil {
			log.Fatalf("[%05d] Failed to send: %s\n", conid, err)
		}
		if l != buflen {
			log.Fatalln("[%05d] Failed to send enough data: %d\n", conid, l)
		}

		// Tell the other end that we are done
		c.CloseWrite()

		w <- l
	}()

	rxbuf := make([]byte, buflen)

	n, err := io.ReadFull(bufio.NewReader(c), rxbuf)
	if err != nil {
		log.Fatalf("[%05d] Failed to receive: %s\n", conid, err)
	}
	csum1 := md5.Sum(rxbuf)

	totalSent := <-w
	fmt.Printf("[%05d] RX: %d bytes, md5=%02x (sent=%d)\n", conid, n, csum1, totalSent)

	if csum0 != csum1 {
		log.Fatalf("[%05d] Checksums don't match", conid)
	}

	message, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		log.Fatalf("[%05d] Failed to receive: %d\n", conid, err)
	}
	fmt.Printf("[%05d] From SVR: %s", conid, message)
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	if serverMode {
		fmt.Printf("Starting server\n")
		server()
	}

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
	fmt.Printf("Client connecting to %s\n", vmid.String())
	for i := 0; i < connections; i++ {
		client(vmid, i)
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}
}
