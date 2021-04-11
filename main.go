package main

import (
	"net"
	"time"
)

var peers map[string]net.Conn
var recentMessages []Message
var localAddress string

var quit chan bool

func main() {
	quit = make(chan bool)
	setupDisplay()

	peers = make(map[string]net.Conn)
	recentMessages = make([]Message, 0)

	// init server and get local addr
	WriteLn(errorMessages, "Initing server...")
	server, err := initServer()
	if err != nil {
		WriteLn(errorMessages, err.Error())
		return
	}
	defer server.Close()

	// have to connect to self to get addr
	c, err := net.Dial("tcp4", server.Addr().String())
	localAddress = c.LocalAddr().String()
	c.Close()

	go listenForConnections(server)

	// was using for loop, but eats up CPU
	select {
	case <-quit:
		return
	}
}

func sendMessage(text string) {
	announceMessage(Message{
		Origin:    localAddress,
		Timestamp: time.Now().String(),
		Data:      text,
	})

	WriteLn(messageText, text)
}

func close() {
	terminalCancel()
	displayTerminal.Close()

	quit <- true
}
