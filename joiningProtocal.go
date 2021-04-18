package main

import (
	"encoding/gob"
	"net"
	"time"
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
	packet := &Packet{}
	err := dec.Decode(packet) // blocking

	if err != nil {
		WriteLn(errorMessages, "error on connection: "+err.Error())
	} else {
		if packet.Type == CONN_REQ {
			handleConnRequest(*packet)
		} else if packet.Type == CONN_ACK {
			handleConnAck(c, *packet)
			return // dont want to close a connection we are adding
		}
	}
	c.Close()
}

func handleConnRequest(packet Packet) {
	if len(peers) >= MAX_PEERS {
		WriteLn(errorMessages, "got connection request, too many clients")
		// pass on request to someone else

		// randomly send
		for peerToPassTo := range peers {
			peersAddr := peerToPassTo.RemoteAddr().String()
			if peersAddr != packet.Origin {
				WriteLn(errorMessages, "passing request to "+peerToPassTo.RemoteAddr().String())
				packet.Data += "-> " + localAddress // if we dont change packet in some way, someone may get a second time and throw away, not what we want
				sendPacket(peerToPassTo, packet)
				break
			}
		}

		// // send to all
		// // creates a fully connected network, quickly no one is available
		// announcePacket(packet)

	} else {
		WriteLn(errorMessages, "got connection request from "+packet.Origin+", connecting then sending ack")
		conn, ok := requestConnection(packet.Origin)
		if ok {
			sendAck(conn)
			addConnectionChan <- conn
		}
	}
}

func handleConnAck(c net.Conn, packet Packet) {
	if len(peers) >= MAX_PEERS {
		WriteLn(errorMessages, "got connection acknowledge, too many clients, ignoring")
		c.Close()
		return
	} else {
		WriteLn(errorMessages, "got connection acknowledge, adding to peers")
		addConnectionChan <- c
	}
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
		if destinationAddr == peer.RemoteAddr().String() {
			WriteLn(errorMessages, "Already connected")
			return nil, false
		}
	}

	c, err := net.Dial("tcp4", destinationAddr)
	if err != nil {
		WriteLn(errorMessages, err.Error())
		return nil, false
	}

	return c, true
}

// adds a connection our list of peers
func addConnection(newConn net.Conn) {
	remoteAddr := newConn.RemoteAddr().String()
	peers[newConn] = true

	WriteLn(errorMessages, "Added connection "+remoteAddr)
	displayPeers(peers)

	go handlePeer(newConn)
}

func sendAck(c net.Conn) {
	ack := Packet{
		Type:      CONN_ACK,
		Origin:    localAddress,
		Timestamp: time.Now().String(),
	}
	sendPacket(c, ack)
}

// creates connection, sends request, then closes. We will get a new connection if someone accepts
// should only be used by a node not connected to any nodes, otherwise send request through peers
func enterNetwork(bootstrapIP string) {
	tmpConn, ok := requestConnection(bootstrapIP)
	if !ok {
		return
	}

	connReq := Packet{
		Type:      CONN_REQ,
		Origin:    localAddress,
		Timestamp: time.Now().String(),
	}
	WriteLn(errorMessages, "sending connection request to "+bootstrapIP)

	sendPacket(tmpConn, connReq)
	tmpConn.Close()
	WriteLn(errorMessages, "closed connection from "+bootstrapIP)
}
