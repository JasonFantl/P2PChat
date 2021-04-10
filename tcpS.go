package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
)

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
	fmt.Println("Listening on", PORT)
	return net.Listen("tcp4", PORT)
}

func listenForConnections(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(c)
	}
}

func connectToPeer() {
	// fmt.Print("Enter IP and port. (i.e. address: 127.0.0.1:1234) \n    address: ")
	// reader := bufio.NewReader(os.Stdin)
	// addr, _ := reader.ReadString('\n')
	// addr = strings.TrimSpace(addr)

	addr := "127.0.0.1:1234"

	c, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	go handleConnection(c)
}

func handleConnection(c net.Conn) {
	addr := c.RemoteAddr().String()
	peers[addr] = c

	fmt.Println("Added connection", addr)

	for {
		dec := gob.NewDecoder(c)
		message := &Message{}
		err := dec.Decode(message)

		if err == io.EOF {
			disconnectFromPeer(addr)
			break
		} else if err != nil {
			fmt.Println(err)
			continue
		}

		recieveMessage(*message)
	}
}

func disconnectFromPeer(addr string) {
	peers[addr].Close()
	delete(peers, addr)
	fmt.Println(addr, "disconnected")
}

func announceMessage(message Message) {

	recentMessages = append(recentMessages, message)

	for _, peer := range peers {
		encoder := gob.NewEncoder(peer)

		// fmt.Println("Sending to", addr)
		err := encoder.Encode(message)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func recieveMessage(message Message) {
	// checking we havent seen this before
	for _, oldMessage := range recentMessages {
		if oldMessage == message {
			// no need to announce we've seen this before
			return
		}
	}

	recentMessages = append(recentMessages, message)
	fmt.Println(message.Origin, ":", message.Data)

	// then pass message on to others
	announceMessage(message)
}
