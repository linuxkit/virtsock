package main

import (
	"hash"
	"log"

	"crypto/md5"
	"math/rand"
)

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
