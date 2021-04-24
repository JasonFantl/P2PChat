package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type PeerMeta struct {
	ConnectionCount int
	GID             string
}

type Peer struct {
	connection net.Conn
	meta       PeerMeta
}

var peers map[*Peer]bool
var recentPackets map[Packet]bool // may want to make into max length queue later
var localAddress string

var quit chan bool
var addPeerChan chan *Peer
var removePeerChan chan *Peer
var waitPeers sync.WaitGroup

var GID string

func main() {

	// init channels and other vars
	quit = make(chan bool)
	addPeerChan = make(chan *Peer)
	removePeerChan = make(chan *Peer)

	peers = make(map[*Peer]bool)
	recentPackets = make(map[Packet]bool)

	// init server and get local addr
	server, err := initServer()
	if err != nil {
		fmt.Printf(err.Error())
		return
	}
	defer server.Close()

	setupDisplay() // after this use WriteLn(errorMessage, string) instead of Println(string)
	defer closeDisplay()

	// have to connect to self to get addr
	c, err := net.Dial("tcp4", server.Addr().String())
	localAddress = c.RemoteAddr().String()
	c.Close()
	WriteLn(errorMessages, "Listening on: "+localAddress)

	GID = localAddress

	go listenForConnections(server)
	WriteLn(errorMessages, "\n")

	// was using for loop, but eats up CPU
	for {
		select {
		case <-quit:
			return
		case newPeer := <-addPeerChan:
			addPeer(newPeer)
			waitPeers.Done()
		case oldPeer := <-removePeerChan:
			removePeer(oldPeer)
			waitPeers.Done()
		}
	}
}

// adds a connection our list of peers
func addPeer(peer *Peer) {
	peers[peer] = true
	displayPeers()
}

// removes a connection from our list of peers
func removePeer(peer *Peer) {
	_, ok := peers[peer]
	if ok {
		delete(peers, peer)
		displayPeers()

		WriteLn(errorMessages, "disconnected, sending out new CONN_REQ")
		connReq := Packet{
			Type:      CONN_REQ,
			Origin:    localAddress,
			Timestamp: time.Now().String(),
		}
		recieveConnectionRequest(connReq)
	}
}

func initServer() (net.Listener, error) {
	fmt.Println("Initing server...")
	arguments := os.Args
	PORT := ":1234"

	if len(arguments) > 1 {
		PORT = ":" + arguments[1]
	}

	return net.Listen("tcp4", PORT)
}

func getMyMeta() PeerMeta {
	return PeerMeta{
		ConnectionCount: len(peers),
		GID:             GID,
	}
}
