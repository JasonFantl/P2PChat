package main

import (
	"net"
	"os"
)

var peers map[net.Conn]bool
var recentPackets map[Packet]bool // may want to make into max length queue later
var localAddress string

var quit chan bool
var addConnectionChan, removePeerChan chan net.Conn
var packetChan chan Packet

func main() {

	// init channels and other vars
	quit = make(chan bool)
	addConnectionChan = make(chan net.Conn)
	removePeerChan = make(chan net.Conn)
	packetChan = make(chan Packet)

	peers = make(map[net.Conn]bool)
	recentPackets = make(map[Packet]bool)

	setupDisplay()

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
	localAddress = c.RemoteAddr().String()
	WriteLn(errorMessages, "obtained local address: "+localAddress)
	c.Close()

	go listenForConnections(server)

	// was using for loop, but eats up CPU
	for {
		select {
		case <-quit:
			terminalCancel()
			displayTerminal.Close()
			return
		case newConn := <-addConnectionChan:
			addConnection(newConn)
		case oldConn := <-removePeerChan:
			removeConnection(oldConn)
		case newPacket := <-packetChan:
			go recievePacket(newPacket)
		}
	}
}

func initServer() (net.Listener, error) {
	arguments := os.Args
	PORT := ":" + arguments[1]
	if len(arguments) == 1 {
		PORT = ":1234"
	}

	WriteLn(errorMessages, "Listening on "+PORT)
	return net.Listen("tcp4", PORT)
}
