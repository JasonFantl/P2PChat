package main

import (
	"encoding/gob"
	"encoding/hex"

	"github.com/jasonfantl/P2PChat/P2Proto"
)

var quit chan bool

type chatroom struct {
	name    string
	key     []byte
	history []string
}

var chatrooms map[string]*chatroom
var currentRoomName string

// func createChatRoom(name string) {
// 	key := make([]byte, 32) //generate a random 32 byte key for AES-256
// 	if _, err := rand.Read(key); err != nil {
// 		panic(err.Error())
// 	}

// 	addChatRoom(name, key)
// }

func addChatRoom(name string, key []byte) {
	if _, exist := chatrooms[name]; exist {
		return
	}

	chatrooms[name] = &chatroom{
		name:    name,
		key:     key,
		history: make([]string, 0),
	}

	// update display
	c.Update(chatID, generateChatLayout()...)
}

type Message []byte

func recievePacket(packet P2Proto.Packet) {
	if packet.Type == P2Proto.MESSAGE {
		message := packet.Payload.(Message)

		for _, chatroom := range chatrooms {
			decrypted, ok := decrypt(message, chatroom.key)
			if !ok {
				continue
			}

			plaintext := string(decrypted)

			text := packet.Origin + ": " + plaintext

			if currentRoomName == chatroom.name {
				WriteLn(messageText, text)
			}
			chatroom.history = append(chatroom.history, text)

		}
	}
}

func sendMessage(plaintext string) error {

	chatroom, ok := chatrooms[currentRoomName]
	if !ok {
		logger("invalid chatroom")
		return nil
	}

	WriteLn(messageText, plaintext)
	chatroom.history = append(chatroom.history, plaintext)

	encrypted := Message(encrypt([]byte(plaintext), chatroom.key))

	P2Proto.SendMessage(encrypted)
	return nil
}

func main() {
	gob.Register(Message{})

	quit = make(chan bool)
	chatrooms = make(map[string]*chatroom)

	setupDisplay()
	defer closeDisplay()

	key, _ := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")
	addChatRoom("test room", key)

	key2, _ := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726575")
	addChatRoom("test room 2", key2)

	updatePeers := func(peers P2Proto.PeerList) {
		displayPeers(peers)
	}

	go P2Proto.Setup(recievePacket, updatePeers, logger)

	for {
		select {
		case <-quit:
			return
		}
	}
}

func logger(s string) {
	WriteLn(errorMessages, s)
}
