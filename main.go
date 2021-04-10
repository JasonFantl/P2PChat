package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var peers map[string]net.Conn
var recentMessages []Message
var addr string

func main() {
	peers = make(map[string]net.Conn)
	recentMessages = make([]Message, 0)

	server, err := initServer()

	if err != nil {
		fmt.Println(err)
		return
	}
	defer server.Close()
	addr = server.Addr().String()

	go listenForConnections(server)
	listenForUserInput()
}

func listenForUserInput() {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		if checkCommand(strings.TrimSpace(text)) {
			break
		}
	}
}

func checkCommand(text string) bool {
	switch text {
	case "STOP":
		return true
	case "CONNECT":
		connectToPeer()
	default:
		sendMessage(text)
	}

	return false
}

func sendMessage(text string) {
	message := Message{
		Origin:    addr,
		Timestamp: time.Now().String(),
		Data:      text,
	}

	announceMessage(message)
}
