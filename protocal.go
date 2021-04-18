package main

import (
	"encoding/gob"
	"io"
	"net"
	"time"
)

type PacketType byte

const (
	MESSAGE PacketType = iota
	CONN_REQ
	CONN_ACK
)

var MAX_PEERS = 2

type Packet struct {
	Type      PacketType
	Origin    string
	Timestamp string // if we use time.Time, then == is false between a packet and a copy of itself, weird
	Data      string
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

	if packet.Type == MESSAGE {
		WriteLn(messageText, packet.Origin+": "+packet.Data)
		announcePacket(packet)
	} else if packet.Type == CONN_REQ {
		handleConnRequest(packet)
	}
}

// removes a connection from our list of peers
func removeConnection(oldConn net.Conn) {
	ok := peers[oldConn]
	if ok {
		WriteLn(errorMessages, oldConn.RemoteAddr().String()+" disconnected")
		oldConn.Close()
		delete(peers, oldConn)
		displayPeers(peers)
	}
}

// sends packet to all peers
func announcePacket(packet Packet) {
	for peer := range peers {
		sendPacket(peer, packet)
	}
}

// sends packet to a peer
func sendPacket(peer net.Conn, packet Packet) {
	recentPackets[packet] = true

	encoder := gob.NewEncoder(peer)
	err := encoder.Encode(packet) // writes to tcp connection

	if err != nil {
		WriteLn(errorMessages, err.Error())
	}
}

// asynchronous function, a different instance is run for each peer
func handlePeer(c net.Conn) {
	for { // wont eat CPU since it has a blocking function in it
		dec := gob.NewDecoder(c)
		packet := &Packet{}
		err := dec.Decode(packet) // blocking till we finish reading message

		if err == io.EOF { // client disconnected
			removePeerChan <- c
			break
		} else if err != nil { // error decoding message
			WriteLn(errorMessages, err.Error())
			continue
		}

		// no errors, handle packet
		packetChan <- *packet
	}
}

// GUI call
func sendMessage(text string) {
	announcePacket(Packet{
		Type:      MESSAGE,
		Origin:    localAddress,
		Timestamp: time.Now().String(),
		Data:      text,
	})

	WriteLn(messageText, text)
}
