package main

import (
	"log"
	"net"
)

const (
	Port       = "9090"
	SafeMode   = true
	BufferSize = 512
)

// Architecture
// We have 1 go routine keeping track of connections
// storing them in clients

// Then another go routine that will continuously be polling
// the outgoing channel for the msgs

func safeRemoteAddr(mode bool, conn net.Conn) string {
	if !mode {
		return "[REDACTED]"
	}

	return conn.RemoteAddr().String()
}

type Client struct {
	conn     net.Conn
	outgoing chan string // we will receive messages thorugh here
}

func server(c chan Client) {
	clients := []Client

}

// Need incoming and outgoing channel to actually send/rcv messages
func handleConnection(conn net.Conn, outgoing chan string) {
	// fixed buffer for messages to be received
	// how is this getting written to? via asynq connection
	buffer := make([]byte, BufferSize)
	// init loop that will listen to messages from user
	for {
		// read from buffer
		// how will we send to all the people listening? need to use channels here
		n, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			return
		}

		outgoing <- string(buffer[0:n])
	}
}

func main() {
	ln, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Fatalf("error: could not listen to port: %s with err: %s\n", Port, err)
	}

	log.Printf("listening to TCP connection on port: %s\n", Port)

	// this is an infinite loop that's listening for connection
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("could not accept connections %s\n", err)
		}

		outgoing := make(chan string)
		log.Printf("accepted connection from remote addr: %s", safeRemoteAddr(SafeMode, conn))
		go handleConnection(conn, outgoing)
	}
}
