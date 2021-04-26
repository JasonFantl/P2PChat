package P2Proto

import (
	"encoding/gob"
	"io"
	"net"
	"time"
)

var MIN_DESIRED_PEERS = 2

type PacketType byte

const (
	MESSAGE PacketType = iota
	CONN_REQ
	CONN_ACK
	BLANK // used to just send meta data, packet is empty
)

type Packet struct {
	Type      PacketType
	Origin    string
	Payload   interface{} // arbitrary data type
	Timestamp string
}

type Carrier struct {
	Packet Packet
	Meta   PeerMeta
}

// asynchronous function, a different instance is run for each peer
func handlePeer(peer *Peer) {

	waitPeers.Add(1)
	addPeerChan <- peer
	waitPeers.Wait()

	log("added connection " + peer.Connection.RemoteAddr().String() + "(" + peer.Meta.GID + ")" + " to peers")

	announceBlank() // to update our neighbors of our new peer count

	// dont return in this loop, have some cleaning up to do afterward
	for {
		dec := gob.NewDecoder(peer.Connection)
		carrier := &Carrier{}
		err := dec.Decode(carrier) // blocking till we finish reading message

		if err == io.EOF { // client disconnected
			break
		} else if err != nil { // error decoding message
			log(err.Error())
			continue
		}

		// no errors, handle packet
		// first update meta about peer
		peer.Meta = carrier.Meta
		alertPeers(Peers)

		recievePacket(carrier.Packet)
	}

	log("stopped handling peer " + peer.Connection.RemoteAddr().String() + "(" + peer.Meta.GID + ")\n")
	peer.Connection.Close()

	waitPeers.Add(1)
	removePeerChan <- peer // update the peer list
	waitPeers.Wait()

	announceBlank() // to update our neighbors of our new peer count
}

func recievePacket(packet Packet) {
	// check we havent seen this packet before (may not always be a good idea, probably have to change later)
	for _, oldPacket := range recentPackets {
		if oldPacket.Timestamp == packet.Timestamp { // should probably have better way of checking this
			return
		}
	}
	// then add it so we dont handle again
	recentPackets = append(recentPackets, packet)

	// make new packet available to handle outside of library
	alertPacket(packet)

	switch packet.Type {
	case MESSAGE:
		recieveMessage(packet)
	case CONN_REQ:
		recieveConnectionRequest(packet)
		// we ignore CONN_ACK since they only act as meta data updters, done in recievePacket func. use this oppertunity to check some stuff
	}
}

func recieveMessage(packet Packet) {
	announcePacket(packet)
}

func recieveConnectionRequest(packet Packet) {
	// double check
	if packet.Type != CONN_REQ {
		log("invalid function call, cannot handle packet not of type CONN_REQ")
		return
	}

	var peerToPassTo *Peer = nil
	// get peer with lowest connection count
	for peer := range Peers {
		if peer.Meta.GID != packet.Origin { // dont pass to the node trying to connect
			if peerToPassTo == nil || peer.Meta.ConnectionCount < peerToPassTo.Meta.ConnectionCount {
				peerToPassTo = peer
			}
		}
	}

	// set to nil if we are the peer with smallest number of connection
	if peerToPassTo != nil && peerToPassTo.Meta.ConnectionCount >= len(Peers) {
		// check we arnt sending the packet for ourselves, then it needs to be sent even if we have the smallest count
		if packet.Origin != localAddress {
			peerToPassTo = nil
		}
	}

	if peerToPassTo == nil {
		if packet.Origin == localAddress {
			log("cannot request connection to self")
		} else {
			log("got connection request from " + packet.Origin + ", accepting")

			conn, ok := requestConnection(packet.Origin)
			if ok {
				newPeer := Peer{
					Connection: conn,
				}
				sendAck(conn) // let them know they are a peer now
				go handlePeer(&newPeer)
			}
		}
	} else {
		log("got connection request from " + packet.Origin + ", forwarding to " + peerToPassTo.Connection.RemoteAddr().String())
		sendPacket(peerToPassTo.Connection, packet)
	}
}

// sends packet to all peers
func announcePacket(packet Packet) {
	for peer := range Peers {
		if peer.Connection.RemoteAddr().String() != packet.Origin {
			sendPacket(peer.Connection, packet)
		}
	}
}

// sends packet to a peer
func sendPacket(connection net.Conn, packet Packet) {
	recentPackets = append(recentPackets, packet)

	// wrap in carrier
	carrier := Carrier{
		Packet: packet,
		Meta:   getMyMeta(),
	}

	encoder := gob.NewEncoder(connection)
	err := encoder.Encode(carrier) // writes to tcp connection

	if err != nil {
		log(err.Error())
	}
}

// GUI call
func SendMessage(payload interface{}) {
	msgPacket := Packet{
		Type:      MESSAGE,
		Origin:    localAddress,
		Payload:   payload,
		Timestamp: time.Now().String(),
	}

	announcePacket(msgPacket)
}
