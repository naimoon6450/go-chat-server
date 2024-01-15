package main

import (
	"log"
	"net"
	"time"
)

const (
	Port       = "9090"
	SafeMode   = true
	UnsafeMode = false
	BufferSize = 512
)

// Architecture
// We have 1 go routine keeping track of connections
// storing them in our conns slice

// Then another go routine that will continuously be polling
// the outgoing channel for the msgs per client

// Questions
// How does nc allow you to continue writing to it?
// It's one of the features of netcat that allows you to send data to the
// connection you're listening to

func safeRemoteAddr(mode bool, conn net.Conn) string {
	if mode {
		return "[REDACTED]"
	}

	return conn.RemoteAddr().String()
}

type MessageType int

const (
	ClientConnected MessageType = iota + 1
	ClientDisconnected
	NewMessage
)

const (
	MsgRate = 1.0
)

type Message struct {
	Type MessageType
	Conn net.Conn
	Text string
}

type Client struct {
	Conn              net.Conn
	LastMessageSentAt time.Time
	Strikes           int
}

func server(msgs chan Message) {
	clients := map[string]*Client{} // making a map of pointers so you can determine if it exists easily via null ptr
	// we want to traverse the slice of c
	// and read the outgoing messages from each one
	for {
		msg := <-msgs
		switch msg.Type {
		case ClientConnected:
			log.Printf("Client %s has connected", safeRemoteAddr(UnsafeMode, msg.Conn))
			clients[msg.Conn.RemoteAddr().String()] = &Client{
				Conn:              msg.Conn,
				LastMessageSentAt: time.Now(), // you can't connect and send a msg instantly
			}
		case ClientDisconnected:
			log.Printf("Deleting Client %s form connection list", safeRemoteAddr(UnsafeMode, msg.Conn))
			// remove from map of connection
			delete(clients, msg.Conn.RemoteAddr().String())
		case NewMessage:
			addr := msg.Conn.RemoteAddr().String()
			now := time.Now()
			clientSendingMsg := clients[addr]
			if now.Sub(clientSendingMsg.LastMessageSentAt).Seconds() >= MsgRate {
				// UPDATE the current clients latest msg sent at time
				clientSendingMsg.LastMessageSentAt = now
				log.Printf("Client %s send message %s", safeRemoteAddr(UnsafeMode, msg.Conn), msg.Text)
				// loop through the rest of the clients to send the msg to them
				for _, client := range clients {
					// do not resend to the author of message
					if client.Conn.RemoteAddr().String() == addr {
						continue
					}

					// write to the output of everyone
					_, err := client.Conn.Write([]byte(msg.Text))
					if err != nil {
						// REMOVE CONN FROM LIST
						// MARK AS DEAD AND CLEAN UP ASYNC?
						log.Println("Could not send data to %s: %s", safeRemoteAddr(UnsafeMode, msg.Conn), err)
					}
				}
			} else {
				clientSendingMsg.Strikes += 1
				if clientSendingMsg.Strikes >= 10 {

				}
			}
		}
	}

}

func client(conn net.Conn, msgQ chan Message) {
	// fixed buffer for messages to be received
	// how is this getting written to? via asynq connection
	buffer := make([]byte, BufferSize)
	// init loop that will listen to messages from user
	for {
		// read from buffer
		// how will we send to all the people listening? need to use channels here
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Could not read from client %s", safeRemoteAddr(UnsafeMode, conn))
			conn.Close()
			msgQ <- Message{
				Type: ClientDisconnected,
				Conn: conn,
			}
			return
		}

		text := string(buffer[0:n])
		msgQ <- Message{
			Type: NewMessage,
			Text: text,
			Conn: conn,
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Fatalf("error: could not listen to port: %s with err: %s\n", Port, err)
	}

	log.Printf("listening to TCP connection on port: %s\n", Port)

	messageQueue := make(chan Message)
	// kick off routine that will accept incoming connections
	// and add to list of clients
	// is also responsible for processing new msgs / sending to everyone
	go server(messageQueue)

	// this is an infinite loop that will ensure
	// clients can send to msg queue
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("could not accept connections %s\n", err)
		}

		log.Printf("accepted connection from remote addr: %s", safeRemoteAddr(UnsafeMode, conn))
		messageQueue <- Message{
			Type: ClientConnected,
			Conn: conn,
		}

		go client(conn, messageQueue)
	}
}
