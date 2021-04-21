package main

import (
	"net"
	"os"
)

type PeerMeta struct {
	ConnectionCount int
	Username        string
}

type Peer struct {
	connection net.Conn
	meta       PeerMeta
}

var peers map[*Peer]bool
var recentPackets map[Packet]bool // may want to make into max length queue later
var localAddress string

var quit chan bool
var addPeerChan, removePeerChan chan *Peer
var packetChan chan Packet

var username string

func main() {

	// init channels and other vars
	quit = make(chan bool)
	addPeerChan = make(chan *Peer)
	removePeerChan = make(chan *Peer)
	packetChan = make(chan Packet)

	peers = make(map[*Peer]bool)
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

	go listenForConnections(server)

	// was using for loop, but eats up CPU
	for {
		select {
		case <-quit:
			terminalCancel()
			displayTerminal.Close()
			return
		case newPeer := <-addPeerChan:
			addPeer(newPeer)

			// // there must be a better place to put this, and a way to make modular, tied to handleReq func
			// // check if we have as many connections as we want
			// if len(peers) < MIN_DESIRED_PEERS {
			// 	var peerToPassTo *Peer = nil
			// 	// get peer with lowest connection count
			// 	for peer := range peers {
			// 		if peerToPassTo == nil || peer.meta.ConnectionCount < peerToPassTo.meta.ConnectionCount {
			// 			peerToPassTo = peer
			// 		}
			// 	}
			// 	sendConnReq(peerToPassTo.connection)
			// }

		case oldPeer := <-removePeerChan:
			removePeer(oldPeer)
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

	username = "user: " + PORT

	WriteLn(errorMessages, "Listening on "+PORT)
	return net.Listen("tcp4", PORT)
}

func getMyMeta() PeerMeta {
	return PeerMeta{
		ConnectionCount: len(peers),
		Username:        username,
	}
}
