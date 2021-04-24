package main

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
	Payload   string
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

	WriteLn(errorMessages, "added connection "+peer.connection.RemoteAddr().String()+"("+peer.meta.GID+")"+" to peers")

	announceBlank() // to update our neighbors of our new peer count

	// dont return in this loop, have some cleaning up to do afterward
	for {
		dec := gob.NewDecoder(peer.connection)
		carrier := &Carrier{}
		err := dec.Decode(carrier) // blocking till we finish reading message

		if err == io.EOF { // client disconnected
			break
		} else if err != nil { // error decoding message
			WriteLn(errorMessages, err.Error())
			continue
		}

		// no errors, handle packet
		// first update meta about peer
		peer.meta = carrier.Meta
		displayPeers()
		recievePacket(carrier.Packet)
	}

	WriteLn(errorMessages, "stopped handling peer "+peer.connection.RemoteAddr().String()+"("+peer.meta.GID+")\n")
	peer.connection.Close()

	waitPeers.Add(1)
	removePeerChan <- peer // update the peer list
	waitPeers.Wait()

	announceBlank() // to update our neighbors of our new peer count
}

func recievePacket(packet Packet) {
	// check we havent seen this packet before (may not always be a good idea, probably have to change later)
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
		// we ignore CONN_ACK since they only act as meta data updters, done in recievePacket func. use this oppertunity to check some stuff
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
		if peer.meta.GID != packet.Origin { // dont pass to the node trying to connect
			if peerToPassTo == nil || peer.meta.ConnectionCount < peerToPassTo.meta.ConnectionCount {
				peerToPassTo = peer
			}
		}
	}
	// set to nil if we are the peer with smallest number of connection
	if peerToPassTo != nil && peerToPassTo.meta.ConnectionCount >= len(peers) {
		// check we arnt sending the packet for ourselves, then it needs to be sent even if we have the smallest count
		if packet.Origin != localAddress {
			peerToPassTo = nil
		}
	}

	if peerToPassTo == nil {
		if packet.Origin == localAddress {
			WriteLn(errorMessages, "cannot request connection to self")
		} else {
			WriteLn(errorMessages, "got connection request from "+packet.Origin+", accepting")

			conn, ok := requestConnection(packet.Origin)
			if ok {
				newPeer := Peer{
					connection: conn,
				}
				sendAck(conn) // let them know they are a peer now
				go handlePeer(&newPeer)
			}
		}
	} else {
		WriteLn(errorMessages, "got connection request from "+packet.Origin+", forwarding to "+peerToPassTo.connection.RemoteAddr().String())
		sendPacket(peerToPassTo.connection, packet)
	}
}

// sends packet to all peers
func announcePacket(packet Packet) {
	for peer := range peers {
		if peer.connection.RemoteAddr().String() != packet.Origin {
			sendPacket(peer.connection, packet)
		}
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
	WriteLn(messageText, text)

	msgPacket := Packet{
		Type:      MESSAGE,
		Origin:    localAddress,
		Payload:   text,
		Timestamp: time.Now().String(),
	}

	announcePacket(msgPacket)
}
