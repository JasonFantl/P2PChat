package main

import (
	"net"
	"os"
)

type PeerMeta struct {
	ConnectionCount int
	GID             string
}

type Peer struct {
	connection net.Conn
	meta       PeerMeta
}

var peers map[string]*Peer
var recentPackets map[Packet]bool // may want to make into max length queue later
var localAddress string

var quit chan bool
var addPeerChan chan *Peer
var removePeerChan chan *Peer

var GID string

func main() {

	// init channels and other vars
	quit = make(chan bool)
	addPeerChan = make(chan *Peer)
	removePeerChan = make(chan *Peer)

	peers = make(map[string]*Peer)
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
	c.Close()
	WriteLn(errorMessages, "obtained local address: "+localAddress)

	GID = localAddress

	go listenForConnections(server)
	WriteLn(errorMessages, "\n")

	// was using for loop, but eats up CPU
	for {
		select {
		case <-quit:
			terminalCancel()
			displayTerminal.Close()
			return
		case newPeer := <-addPeerChan:
			addPeer(newPeer)
		case oldPeer := <-removePeerChan:
			removePeer(oldPeer)
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

func getMyMeta() PeerMeta {
	return PeerMeta{
		ConnectionCount: len(peers),
		GID:             GID,
	}
}
