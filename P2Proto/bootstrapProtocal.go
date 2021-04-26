package P2Proto

import (
	"encoding/gob"
	"io"
	"net"
	"time"
)

// listens for new TCP connections
func listenForConnections(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			log(err.Error())
			continue
		}
		go handleConnection(c)
	}
}

// asynchronous function, for tmp connections before they become peers. only accepts one packet, then closes
func handleConnection(conn net.Conn) {
	log("handling connection: " + conn.RemoteAddr().String())

	dec := gob.NewDecoder(conn)
	carrier := &Carrier{}
	err := dec.Decode(carrier) // blocking till we finish reading message

	if err == io.EOF { // client disconnected
		// do nothing, closing connection later
	} else if err != nil { // error decoding message
		log(err.Error())
	} else { // no errors, handle packet
		switch carrier.Packet.Type {
		case CONN_REQ:
			recieveConnectionRequest(carrier.Packet)
		case CONN_ACK:
			recieveConnectionAcknowledgment(conn, *carrier)
			return // we have handlePeer that deals with closing the connection now
		}
	}

	log("stopped handling connection: " + conn.RemoteAddr().String() + "\n")
	conn.Close()
}

func recieveConnectionAcknowledgment(conn net.Conn, carrier Carrier) {
	// double check
	if carrier.Packet.Type != CONN_ACK {
		log("invalid function call, cannot handle packet not of type CONN_ACK")
		return
	}

	log("got connection acknowledge from " + conn.RemoteAddr().String())

	for peer := range Peers {
		if carrier.Meta.GID == peer.Meta.GID {
			log("already connected to " + conn.RemoteAddr().String() + "(" + carrier.Meta.GID + ")")
			return
		}
	}

	newPeer := Peer{
		Connection: conn,
		Meta:       carrier.Meta,
	}
	handlePeer(&newPeer)
}

// creates the connection to a machine
func requestConnection(destinationAddr string) (net.Conn, bool) {
	log("requesting connection to " + destinationAddr)
	// verify you can connect
	if destinationAddr == localAddress {
		log("Cannot connect to yourself")
		return nil, false
	}
	for peer := range Peers {
		if destinationAddr == peer.Connection.RemoteAddr().String() { // FIX THIS, listening and writting port are different, may not the addrs as the same
			log("Already connected to " + destinationAddr)
			return nil, false
		}
	}

	conn, err := net.Dial("tcp4", destinationAddr)
	if err != nil {
		log(err.Error())
		return nil, false
	}
	log("connection established with " + destinationAddr)
	return conn, true
}

// creates connection, sends request, then closes. We will get a new connection if someone accepts
// should only be used by a node not connected to any nodes, otherwise send request through peers
func EnterNetwork(bootstrapIP string) {
	tmpConn, ok := requestConnection(bootstrapIP)
	if !ok {
		return
	}
	sendConnReq(tmpConn)
	tmpConn.Close()
	log("closed connection to " + bootstrapIP + "\n")
}

func sendAck(c net.Conn) {
	log("sending CONN_ACK to " + c.RemoteAddr().String())
	ack := Packet{
		Type: CONN_ACK,
		// ACK origin is recognized by the connetion it came over, no need for origin field
		Timestamp: time.Now().String(),
	}
	sendPacket(c, ack)
}

func sendConnReq(c net.Conn) {
	log("sending CONN_REQ to " + c.RemoteAddr().String())
	connReq := Packet{
		Type:      CONN_REQ,
		Origin:    localAddress,
		Timestamp: time.Now().String(),
	}
	sendPacket(c, connReq)
}

func announceBlank() {
	log("announcing BLANK")
	blank := Packet{
		Type: BLANK,
	}
	announcePacket(blank)
}
