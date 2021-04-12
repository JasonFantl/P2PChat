package main

import (
	"errors"
	"net"
	"os"
)

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
		newConnectionChan <- c
	}
}

func requestConnection(destinationAddr string) {

	// verify you can connect
	if destinationAddr == localAddress {
		WriteLn(errorMessages, "Cannot connect to yourslf")
		return
	}
	for peer := range peers {
		if destinationAddr == peer.RemoteAddr().String() {
			WriteLn(errorMessages, "Already connected")
			return
		}
	}

	c, err := net.Dial("tcp4", destinationAddr)
	if err != nil {
		WriteLn(errorMessages, err.Error())
		return
	}

	newConnectionChan <- c
}

func removeConnection(oldConn net.Conn) {
	ok := peers[oldConn]
	if ok {
		WriteLn(errorMessages, oldConn.RemoteAddr().String()+" disconnected")
		oldConn.Close()
		delete(peers, oldConn)
		displayPeers(peers)
	}
}

func addConnection(newConn net.Conn) {
	remoteAddr := newConn.RemoteAddr().String()
	peers[newConn] = true

	WriteLn(errorMessages, "Added connection "+remoteAddr)
	displayPeers(peers)

	go handleConnection(newConn)
}
