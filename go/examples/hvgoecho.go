package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

var (
	clientStr  string
	serverMode bool
	verbose    int
)

type Conn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

type Client interface {
	String() string
	Dial(conid int) (Conn, error)
}

func init() {
	flag.StringVar(&clientStr, "c", "", "Client")
	flag.BoolVar(&serverMode, "s", false, "Start as a Server")
	flag.IntVar(&verbose, "v", 0, "Set the verbosity level")
}

func server() {
	l := ServerListen()
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
		log.Fatalln("Copy():", err)
	}
	fmt.Printf("Copied Bytes: %d\n", n)

	fmt.Printf("Sending BYE message\n")
	// The '\n' is important as the client use ReadString()
	_, err = fmt.Fprintf(c, "Got %d bytes. Bye\n", n)
	if err != nil {
		log.Fatalln("Failed to send: ", err)
	}
	fmt.Printf("Sent bye\n")
}

func client(cl Client) {
	c, err := cl.Dial(0)
	if err != nil {
		log.Fatalln("Failed to Dial:\n", cl.String(), err)
	}

	defer func() {
		fmt.Printf("Closing\n")
		c.Close()
	}()

	fmt.Printf("Send: hello\n")
	// Note the '\n' is significant as we use ReadString below
	l, err := fmt.Fprintf(c, "hello\n")
	if err != nil {
		log.Fatalln("Failed to send: ", err)
	}
	fmt.Printf("Sent: %d bytes\n", l)

	reader := bufio.NewReader(c)
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalln("Failed to receive: ", err)
	}
	fmt.Printf("From SVR: %s", message)

	fmt.Printf("CloseWrite()\n")
	c.CloseWrite()

	fmt.Printf("Waiting for Bye message\n")
	message, err = reader.ReadString('\n')
	if err != nil {
		log.Fatalln("Failed to receive: ", err)
	}
	fmt.Printf("From SVR: %s", message)
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	ValidateOptions()

	if serverMode {
		fmt.Printf("Starting server\n")
		server()
	}

	cl := ParseClientStr(clientStr)

	fmt.Printf("Client connecting to %s", cl.String())
	client(cl)
}
