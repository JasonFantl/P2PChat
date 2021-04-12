package main

import (
	"net"
)

var peers map[net.Conn]bool
var recentPackets []Packet
var localAddress string

var quit chan bool
var newConnectionChan, removeConnectionChan chan net.Conn

func main() {
	quit = make(chan bool)
	newConnectionChan = make(chan net.Conn)
	removeConnectionChan = make(chan net.Conn)

	setupDisplay()

	peers = make(map[net.Conn]bool)
	recentPackets = make([]Packet, 0)

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
	for {
		select {
		case <-quit:
			terminalCancel()
			displayTerminal.Close()
			return
		case newConn := <-newConnectionChan:
			addConnection(newConn)
		case oldConn := <-removeConnectionChan:
			removeConnection(oldConn)
		}
	}
}
