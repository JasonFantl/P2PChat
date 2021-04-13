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
)

// Packet is what is passed over TCP
type Packet struct {
	Type      PacketType
	Origin    string
	Timestamp string // if we use time.Time, then == is false betweem a packet and a copy of itself, weird
	Data      string
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

func recievePacket(packet Packet) {
	for _, oldPacket := range recentPackets {
		if oldPacket == packet {
			return
		}
	}

	WriteLn(messageText, packet.Origin+": "+packet.Data)

	announcePacket(packet)
}

func sendMessage(text string) {
	announcePacket(Packet{
		Type:      MESSAGE,
		Origin:    localAddress,
		Timestamp: time.Now().String(),
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
