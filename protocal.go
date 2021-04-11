package main

import (
	"encoding/gob"
	"errors"
	"io"
	"net"
	"os"
)

// Message is what is passed over TCP
type Message struct {
	Origin    string
	Timestamp string
	Data      string
}

func initServer() (net.Listener, error) {
	arguments := os.Args
	if len(arguments) == 1 {
		return nil, errors.New("Port required to open server")
	}

	PORT := ":" + arguments[1]
	WriteLn(errorMessages, "Listening on "+PORT)
	return net.Listen("tcp4", PORT)
}

func listenForConnections(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			WriteLn(errorMessages, err.Error())
			continue
		}
		go handleConnection(c)
	}
}

func connectToPeer(ip, port string) {
	addr := ip + ":" + port

	c, err := net.Dial("tcp", addr)
	if err != nil {
		WriteLn(errorMessages, err.Error())
		return
	}

	go handleConnection(c)
}

func handleConnection(c net.Conn) {
	addr := c.RemoteAddr().String()
	peers[addr] = c

	WriteLn(errorMessages, "Added connection "+addr)

	for {
		dec := gob.NewDecoder(c)
		message := &Message{}
		err := dec.Decode(message)

		if err == io.EOF {
			disconnectFromPeer(addr)
			break
		} else if err != nil {
			WriteLn(errorMessages, err.Error())
			continue
		}

		recieveMessage(*message)
	}
}

func disconnectFromPeer(addr string) {
	peers[addr].Close()
	delete(peers, addr)
	WriteLn(errorMessages, addr+" disconnected")
}

func announceMessage(message Message) {

	recentMessages = append(recentMessages, message)

	for _, peer := range peers {
		encoder := gob.NewEncoder(peer)
		err := encoder.Encode(message)

		if err != nil {
			WriteLn(errorMessages, err.Error())
		}
	}
}

// !!! NOTE: this is not currently concurrent safe, but being used concurrently
func recieveMessage(message Message) {
	for _, oldMessage := range recentMessages {
		if oldMessage == message {
			return
		}
	}

	recentMessages = append(recentMessages, message)

	WriteLn(messageText, message.Origin+": "+message.Data)

	announceMessage(message)
}
