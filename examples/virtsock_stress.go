package main

import (
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"crypto/md5"
	"math/rand"
	"sync/atomic"
)

const (
	ioTimeout = 60 * time.Second
)

var (
	clientStr   string
	serverMode  bool
	maxDataLen  int
	minDataLen  int
	minBufLen   int
	maxBufLen   int
	connections int
	sleepTime   int
	verbose     int
	exitOnError bool
	parallel    int

	connCounter int32
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
	flag.IntVar(&minDataLen, "L", 0, "Minimum Length of data")
	flag.IntVar(&maxDataLen, "l", 64*1024, "Maximum Length of data")
	flag.IntVar(&minBufLen, "B", 16*1024, "Minimum Buffer size")
	flag.IntVar(&maxBufLen, "b", 16*1024, "Maximum Buffer size")
	flag.IntVar(&connections, "i", 100, "Total number of connections")
	flag.IntVar(&sleepTime, "w", 0, "Sleep time in seconds between new connections")
	flag.IntVar(&parallel, "p", 1, "Run n connections in parallel")
	flag.BoolVar(&exitOnError, "e", false, "Exit when an error occurs")
	flag.IntVar(&verbose, "v", 0, "Set the verbosity level")

	rand.Seed(time.Now().UnixNano())
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	SetVerbosity()
	ValidateOptions()

	if serverMode {
		fmt.Printf("Starting server\n")
		server()
		return
	}

	if minDataLen > maxDataLen {
		fmt.Printf("minDataLen > maxDataLen!")
		return
	}
	if minBufLen > maxBufLen {
		fmt.Printf("minBuflen > maxBufLen!")
		return
	}

	cl := ParseClientStr(clientStr)

	if parallel <= 1 {
		// No parallelism, run in the main thread.
		fmt.Printf("Client connecting to %s\n", cl.String())
		for i := 0; i < connections; i++ {
			client(cl, i)
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
		return
	}

	// Parallel clients
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go parClient(&wg, cl)
	}
	wg.Wait()
}

func server() {
	l := ServerListen()
	defer func() {
		l.Close()
	}()

	connid := 0

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Accept(): %s\n", err)
		}

		prDebug("[%05d] accept(): %s -> %s \n", connid, conn.RemoteAddr(), conn.LocalAddr())
		go handleRequest(conn, connid)
		connid++
	}
}

func handleRequest(c net.Conn, connid int) {
	defer func() {
		prDebug("[%05d] Closing\n", connid)
		err := c.Close()
		if err != nil {
			prError("[%05d] Close(): %s\n", connid, err)
		}
	}()

	n, err := io.Copy(c, c)
	if err != nil {
		prError("[%05d] Copy(): %s", connid, err)
		return
	}
	prInfo("[%05d] Copied Bytes: %d\n", connid, n)

	if n == 0 {
		return
	}
}

func parClient(wg *sync.WaitGroup, cl Client) {
	connid := int(atomic.AddInt32(&connCounter, 1))
	for connid < connections {
		client(cl, connid)
		connid = int(atomic.AddInt32(&connCounter, 1))
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}

	wg.Done()
}

func md5Hash(h hash.Hash) [16]byte {
	if h.Size() != md5.Size {
		log.Fatalln("Hash is not an md5!")
	}
	s := h.Sum(nil) // Gets a slice

	var r [16]byte

	for i, b := range s {
		r[i] = b
	}
	return r

}

func client(cl Client, conid int) {
	c, err := cl.Dial(conid)
	if c == nil {
		prError("[%05d] Failed to Dial: %s %s\n", conid, cl, err)
		return
	}

	defer c.Close()

	// Create buffer with random data and random length.
	// Make sure the buffer is not zero-length
	buflen := minDataLen
	if maxDataLen > minDataLen {
		buflen += rand.Intn(maxDataLen - minDataLen + 1)
	}
	hash0 := md5.New()

	var txTime, rxTime time.Duration
	start := time.Now()

	w := make(chan int)
	go func() {
		total := 0
		remaining := buflen
		for remaining > 0 {
			batch := 0
			bufsize := minBufLen
			if maxBufLen > minBufLen {
				bufsize += rand.Intn(maxBufLen - minBufLen + 1)
			}
			if remaining > bufsize {
				batch = bufsize
			} else {
				batch = remaining
			}

			txbuf := randBuf(batch)
			hash0.Write(txbuf)

			e := make(chan error, 0)
			go func() {
				l, err := c.Write(txbuf)
				if err != nil {
					e <- err
				} else if l != batch {
					e <- fmt.Errorf("Sent incorrect length: expected %d, got %d", batch, l)
				} else {
					e <- nil
				}
			}()

			select {
			case err := <-e:
				if err != nil {
					prError("[%05d] Failed to send: %s\n", conid, err)
					break
				}
			case <-time.After(ioTimeout):
				prError("[%05d] Send timed out\n", conid)
				break
			}

			total += batch
			remaining -= batch
		}

		// Tell the other end that we are done
		c.CloseWrite()

		txTime = time.Since(start)
		w <- total
	}()

	hash1 := md5.New()

	totalReceived := 0
	remaining := buflen
	for remaining > 0 {
		batch := 0
		bufsize := minBufLen
		if maxBufLen > minBufLen {
			bufsize += rand.Intn(maxBufLen - minBufLen + 1)
		}
		if remaining > bufsize {
			batch = bufsize
		} else {
			batch = remaining
		}

		rxbuf := make([]byte, batch)

		e := make(chan error, 0)
		go func() {
			l, err := io.ReadFull(c, rxbuf)
			if err != nil {
				e <- err
			} else if l != batch {
				e <- fmt.Errorf("Received incorrect length, expected %d, got %d", batch, l)
			} else {
				e <- nil
			}
		}()

		select {
		case err := <-e:
			if err != nil {
				prError("[%05d] Failed to receive: %s\n", conid, err)
				break
			}
		case <-time.After(ioTimeout):
			prError("[%05d] Receive timed out\n", conid)
			break
		}

		hash1.Write(rxbuf)
		remaining -= batch
		totalReceived += batch
	}

	rxTime = time.Since(start)
	totalSent := <-w

	csum0 := md5Hash(hash0)
	prDebug("[%05d] TX: %d bytes, md5=%02x in %s\n", conid, totalSent, csum0, txTime)

	csum1 := md5Hash(hash1)
	prInfo("[%05d] RX: %d bytes, md5=%02x in %s (sent=%d)\n", conid, totalReceived, csum1, rxTime, totalSent)
	if csum0 != csum1 {
		prError("[%05d] Checksums don't match", conid)
	}
}

func randBuf(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func prError(format string, args ...interface{}) {
	if exitOnError {
		log.Fatalf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func prInfo(format string, args ...interface{}) {
	if verbose > 0 {
		log.Printf(format, args...)
	}
}

func prDebug(format string, args ...interface{}) {
	if verbose > 1 {
		log.Printf(format, args...)
	}
}
