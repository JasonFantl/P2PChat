package main

import (
	"encoding/gob"
	"io"
	"net"
)

var MIN_DESIRED_PEERS = 2

type PacketType byte

const (
	MESSAGE PacketType = iota
	CONN_REQ
	CONN_ACK
)

type Packet struct {
	Type    PacketType
	Origin  string
	Payload string
}

type Carrier struct {
	Packet Packet
	Meta   PeerMeta
}

// asynchronous function, a different instance is run for each peer
func handlePeer(peer *Peer) {
	for { // wont eat CPU since it has a blocking function in it
		dec := gob.NewDecoder(peer.connection)
		carrier := &Carrier{}
		err := dec.Decode(carrier) // blocking till we finish reading message

		if err == io.EOF { // client disconnected
			removePeerChan <- peer
			break
		} else if err != nil { // error decoding message
			WriteLn(errorMessages, err.Error())
			continue
		}

		// no errors, handle packet
		// first update meta about peer
		peer.meta = carrier.Meta
		displayPeers()
		packetChan <- carrier.Packet
	}
}

// called as a go routine
func recievePacket(packet Packet) {

	// check we havent seen this packet before (may not always be true, probably have to change later)
	for oldPacket := range recentPackets {
		if oldPacket == packet {
			return
		}
	}
	// then add it so we dont handle again
	recentPackets[packet] = true

	switch packet.Type {
	case MESSAGE:
		recieveMessage(packet)
	case CONN_REQ:
		recieveConnectionRequest(packet)
		// we ignore CONN_ACK since they only act as meta data updters, done in recievePacket func
	}
}

func recieveMessage(packet Packet) {
	WriteLn(messageText, packet.Origin+": "+packet.Payload)
	announcePacket(packet)
}

func recieveConnectionRequest(packet Packet) {
	// double check
	if packet.Type != CONN_REQ {
		WriteLn(errorMessages, "invalid function call, cannot handle packet not of type CONN_REQ")
		return
	}

	var peerToPassTo *Peer = nil
	// get peer with lowest connection count
	for peer := range peers {
		if peerToPassTo == nil || peer.meta.ConnectionCount < peerToPassTo.meta.ConnectionCount {
			peerToPassTo = peer
		}
	}
	// set to nil if we are the peer with smallest number of connection
	if peerToPassTo != nil && peerToPassTo.meta.ConnectionCount >= len(peers) {
		peerToPassTo = nil
	}

	if peerToPassTo == nil {
		WriteLn(errorMessages, "got connection request from "+packet.Origin+", connecting")
		conn, ok := requestConnection(packet.Origin)
		if ok {
			newPeer := Peer{
				connection: conn,
			}
			addPeerChan <- &newPeer
		}
	} else {
		WriteLn(errorMessages, "got connection request from "+packet.Origin+", forwarding to "+peerToPassTo.connection.RemoteAddr().String())
		sendPacket(peerToPassTo.connection, packet)
	}
}

// removes a connection from our list of peers
func removePeer(peer *Peer) {
	ok := peers[peer]
	if ok {
		WriteLn(errorMessages, peer.connection.RemoteAddr().String()+" disconnected")
		peer.connection.Close()
		delete(peers, peer)
		displayPeers()

		// // then try to make a new connection
		// connReq := Packet{
		// 	Type:      CONN_REQ,
		// 	Origin:    localAddress,
		// 	Timestamp: time.Now().String(),
		// }
		// // send to random peer
		// for peerToPassTo := range peers {
		// 	WriteLn(errorMessages, "passing request to "+peerToPassTo.RemoteAddr().String())
		// 	sendPacket(peerToPassTo, connReq)
		// 	break
		// }
	}
}

// sends packet to all peers
func announcePacket(packet Packet) {
	for peer := range peers {
		sendPacket(peer.connection, packet)
	}
}

// sends packet to a peer
func sendPacket(connection net.Conn, packet Packet) {
	recentPackets[packet] = true

	// wrap in carrier
	carrier := Carrier{
		Packet: packet,
		Meta:   getMyMeta(),
	}

	encoder := gob.NewEncoder(connection)
	err := encoder.Encode(carrier) // writes to tcp connection

	if err != nil {
		WriteLn(errorMessages, err.Error())
	}
}

// GUI call
func sendMessage(text string) {

	msgPacket := Packet{
		Type:    MESSAGE,
		Origin:  localAddress,
		Payload: text,
	}

	announcePacket(msgPacket)

	WriteLn(messageText, text)
}
