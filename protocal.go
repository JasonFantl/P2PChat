package main

import (
	"encoding/gob"
	"io"
	"net"
	"time"
)

type MessageType byte

const (
	MESSAGE MessageType = iota
	CONN_REQ
)

// Packet is what is passed over TCP
type Packet struct {
	Type      MessageType
	Origin    net.IP
	Timestamp time.Time
	Data      string
}

func (p1 Packet) equals(p2 Packet) bool {
	return p1.Type == p2.Type &&
		p1.Origin.Equal(p2.Origin) &&
		p1.Timestamp.Equal(p2.Timestamp) &&
		p1.Data == p2.Data
}

func handleConnection(c net.Conn) {

	for { // wont eat CPU since it has a blocking function in it
		dec := gob.NewDecoder(c)
		message := &Packet{}
		err := dec.Decode(message) // blocking till we finish reading message

		if err == io.EOF { // client dissconnected
			removeConnectionChan <- c
			break
		} else if err != nil { // error decoding message
			WriteLn(errorMessages, err.Error())
			continue
		}

		// no errors, handle message
		recievePacket(*message)
	}
}

func sendMessage(text string) {
	announcePacket(Packet{
		Type:      MESSAGE,
		Origin:    net.ParseIP(localAddress),
		Timestamp: time.Now(),
		Data:      text,
	})

	WriteLn(messageText, text)
}

func announcePacket(packet Packet) {

	recentPackets = append(recentPackets, packet)

	for peer := range peers {
		encoder := gob.NewEncoder(peer)
		err := encoder.Encode(packet) // writes to tcp connection

		if err != nil {
			WriteLn(errorMessages, err.Error())
		}
	}
}

func recievePacket(packet Packet) {
	for _, oldPacket := range recentPackets {
		if oldPacket.equals(packet) {
			return
		}
	}

	WriteLn(messageText, packet.Origin.String()+": "+packet.Data)

	announcePacket(packet)
}
