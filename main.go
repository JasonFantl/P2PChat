package main

import (
	"net"
	"time"
)

var peers map[string]net.Conn
var recentMessages []Message
var addr string

var running = true

func main() {
	setupDisplay()

	peers = make(map[string]net.Conn)
	recentMessages = make([]Message, 0)

	WriteLn(errorMessages, "Initing server...")
	server, err := initServer()

	if err != nil {
		WriteLn(errorMessages, err.Error())
		return
	}
	defer server.Close()
	addr = server.Addr().String()

	go listenForConnections(server)

	for running {

	}
}

func sendMessage(text string) {
	announceMessage(Message{
		Origin:    addr,
		Timestamp: time.Now().String(),
		Data:      text,
	})

	WriteLn(messageText, text)
}

func close() {
	terminalCancel()
	displayTerminal.Close()

	running = false
}
