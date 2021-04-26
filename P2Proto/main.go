package P2Proto

import (
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
	Connection net.Conn
	Meta       PeerMeta
}

type PeerList map[*Peer]bool

var Peers PeerList
var recentPackets []Packet
var localAddress string

var addPeerChan chan *Peer
var removePeerChan chan *Peer

var waitPeers sync.WaitGroup

var GID string

// fuunctions to update outside library
var log func(string)
var alertPacket func(Packet)
var alertPeers func(PeerList)

// blocks, should be called as a go routine
func Setup(p func(Packet), u func(PeerList), l func(string)) {
	// init channels and other vars
	log = l
	alertPacket = p
	alertPeers = u

	addPeerChan = make(chan *Peer)
	removePeerChan = make(chan *Peer)

	Peers = make(map[*Peer]bool)
	recentPackets = make([]Packet, 0)

	// init server and get local addr
	server, err := initServer()
	if err != nil {
		log(err.Error())
		return
	}
	defer server.Close()

	// have to connect to self to get addr
	c, err := net.Dial("tcp4", server.Addr().String())
	localAddress = c.RemoteAddr().String()
	c.Close()
	log("Listening on: " + localAddress)

	GID = localAddress

	go listenForConnections(server)
	log("\n")

	// was using for loop, but eats up CPU
	for {
		select {
		case newPeer := <-addPeerChan:
			Peers[newPeer] = true

			alertPeers(Peers)
			waitPeers.Done()
		case oldPeer := <-removePeerChan:
			_, ok := Peers[oldPeer]
			if ok {
				delete(Peers, oldPeer)

				log("disconnected, sending out new CONN_REQ")
				connReq := Packet{
					Type:      CONN_REQ,
					Origin:    localAddress,
					Timestamp: time.Now().String(),
				}
				recieveConnectionRequest(connReq)
			}

			alertPeers(Peers)
			waitPeers.Done()
		}
	}
}

func initServer() (net.Listener, error) {
	log("Initing server...")
	arguments := os.Args
	PORT := ":1234"

	if len(arguments) > 1 {
		PORT = ":" + arguments[1]
	}

	return net.Listen("tcp4", PORT)
}

func getMyMeta() PeerMeta {
	return PeerMeta{
		ConnectionCount: len(Peers),
		GID:             GID,
	}
}
