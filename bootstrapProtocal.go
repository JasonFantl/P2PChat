package main

import (
	"encoding/gob"
	"io"
	"net"
)

// listens for new TCP connections
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

// asynchronous function, for tmp connections before they become peers. only accepts one packet
func handleConnection(c net.Conn) {
	WriteLn(errorMessages, "handling connection: "+c.RemoteAddr().String())

	dec := gob.NewDecoder(c)
	carrier := &Carrier{}
	err := dec.Decode(carrier) // blocking till we finish reading message

	if err == io.EOF { // client disconnected
		// do nothing, closing connection later
	} else if err != nil { // error decoding message
		WriteLn(errorMessages, err.Error())
	} else { // no errors, handle packet
		switch carrier.Packet.Type {
		case CONN_REQ:
			recieveConnectionRequest(carrier.Packet)
		case CONN_ACK:
			recieveConnectionAcknowledgment(c, *carrier)
			return // dont want to close a connection we are adding
		}
	}

	WriteLn(errorMessages, "stopped handling connection: "+c.RemoteAddr().String())
	c.Close()
}

func recieveConnectionAcknowledgment(conn net.Conn, carrier Carrier) {
	// double check
	if carrier.Packet.Type != CONN_ACK {
		WriteLn(errorMessages, "invalid function call, cannot handle packet not of type CONN_ACK")
		return
	}

	WriteLn(errorMessages, "got connection acknowledge, adding to peers")

	newPeer := Peer{
		connection: conn,
		meta:       carrier.Meta,
	}
	addPeerChan <- &newPeer
}

// creates the connection to a machine
func requestConnection(destinationAddr string) (net.Conn, bool) {
	WriteLn(errorMessages, "requesting connection to "+destinationAddr)
	// verify you can connect
	if destinationAddr == localAddress {
		WriteLn(errorMessages, "Cannot connect to yourself")
		return nil, false
	}
	for peer := range peers {
		if destinationAddr == peer.connection.RemoteAddr().String() {
			WriteLn(errorMessages, "Already connected")
			return nil, false
		}
	}

	conn, err := net.Dial("tcp4", destinationAddr)
	if err != nil {
		WriteLn(errorMessages, err.Error())
		return nil, false
	}

	return conn, true
}

// adds a connection our list of peers
func addPeer(peer *Peer) {
	peers[peer] = true

	WriteLn(errorMessages, "Added connection "+peer.connection.RemoteAddr().String()+" to peers, sending ACK")
	sendAck(peer.connection)
	displayPeers()

	go handlePeer(peer)
}

// creates connection, sends request, then closes. We will get a new connection if someone accepts
// should only be used by a node not connected to any nodes, otherwise send request through peers
func enterNetwork(bootstrapIP string) {
	tmpConn, ok := requestConnection(bootstrapIP)
	if !ok {
		return
	}

	WriteLn(errorMessages, "sending connection request to "+bootstrapIP)
	sendConnReq(tmpConn)
	tmpConn.Close()
	WriteLn(errorMessages, "closed connection from "+bootstrapIP)
}

func sendAck(c net.Conn) {
	ack := Packet{
		Type:   CONN_ACK,
		Origin: localAddress,
	}
	sendPacket(c, ack)
}

func sendConnReq(c net.Conn) {
	WriteLn(errorMessages, "Sending out CONN_REQ to "+c.RemoteAddr().String())
	connReq := Packet{
		Type:   CONN_REQ,
		Origin: localAddress,
	}
	sendPacket(c, connReq)
}