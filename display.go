package main

import (
	"context"
	"flag"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/jasonfantl/P2PChat/P2Proto"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/button"
	"github.com/mum4k/termdash/widgets/text"
	"github.com/mum4k/termdash/widgets/textinput"
)

var textInput *textinput.TextInput
var messageText *text.Text
var errorMessages *text.Text
var peersList *text.Text
var connectField *textinput.TextInput

var c *container.Container

func messagingInput() (*textinput.TextInput, error) {
	input, err := textinput.New(
		textinput.Label("Message: ", cell.FgColor(cell.ColorSilver)),
		textinput.FillColor(cell.ColorDefault),
		textinput.Border(linestyle.Double),
		textinput.OnSubmit(sendMessage),
		textinput.ClearOnSubmit(),
	)
	if err != nil {
		return nil, err
	}
	return input, err
}

func ConnectInput() (*textinput.TextInput, error) {
	input, err := textinput.New(
		textinput.Label("Connect: ", cell.FgColor(cell.ColorCyan)),
		textinput.FillColor(cell.ColorGray),
		textinput.OnSubmit(func(text string) error {
			if text == "" {
				P2Proto.EnterNetwork("127.0.0.1:1234")
			} else {
				P2Proto.EnterNetwork(text)
			}
			return nil
		}),
		textinput.ClearOnSubmit(),
	)
	if err != nil {
		return nil, err
	}
	return input, err
}

func newWrappedRollingText() (*text.Text, error) {
	t, err := text.New(text.RollContent(), text.WrapAtWords())
	if err != nil {
		return nil, err
	}

	return t, nil
}

var chatID = "chatID"
var messageTextID = "messagesID"

func newChatroomButton(name string) (*button.Button, error) {
	opts := []button.Option{
		// button.WidthFor("sparklines"),
		button.FillColor(cell.ColorNumber(220)),
		button.Height(1),
		button.DisableShadow(),
	}

	newButton, err := button.New(name, func() error {
		currentRoomName = name
		return c.Update(messageTextID, generateMessageLayout()...)
	}, opts...)
	if err != nil {
		return nil, err
	}

	return newButton, nil
}

func WriteLn(t *text.Text, text string) {
	err := t.Write(text + "\n")
	if err != nil {
		t.Write(err.Error() + "\n")
	}
}

func displayPeers(peers P2Proto.PeerList) {
	peersList.Reset()

	var ips []string
	for peer := range peers {
		ips = append(ips, peer.Meta.GID+" "+strconv.Itoa(peer.Meta.ConnectionCount))
	}
	sort.Strings(ips)

	for _, ip := range ips {
		WriteLn(peersList, ip)
	}
}

func generateMessageLayout() []container.Option {

	chatroom, ok := chatrooms[currentRoomName]
	if ok {
		messageText.Reset()
		for _, h := range chatroom.history {
			WriteLn(messageText, h)
		}
	}

	return []container.Option{
		container.Border(linestyle.Light),
		container.BorderTitle(currentRoomName),
		container.PlaceWidget(messageText),
		container.ID(messageTextID),
	}
}

func generateChatLayout() []container.Option {

	layout := []container.Option{}

	keys := make([]string, 0)
	for k, _ := range chatrooms {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		b, err := newChatroomButton(name)
		if err != nil {
			continue
		}
		layout = []container.Option{
			container.SplitHorizontal(
				container.Top(
					container.PlaceWidget(b),
				),
				container.Bottom(
					layout...,
				),
				container.SplitFixed(2),
			),
		}
	}

	return append(layout,
		container.BorderTitle("Chatrooms:"),
		container.Border(linestyle.Double),
		container.ID(chatID),
	)
}

func contLayout() ([]container.Option, error) {
	messagingLayout := []container.Option{
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(textInput),
				container.PaddingRight(10),
				container.PaddingLeft(2),
			),
			container.Bottom(generateMessageLayout()...),
			container.SplitFixed(3),
		),
	}

	chatroomsLayout := generateChatLayout()

	mainLayout := []container.Option{
		container.SplitVertical(
			container.Left(chatroomsLayout...),
			container.Right(messagingLayout...),
			container.SplitPercent(20),
		),
	}

	peersLayout := []container.Option{
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(connectField),
				container.PaddingRightPercent(10),
				container.PaddingLeftPercent(10),
			),
			container.Bottom(
				container.Border(linestyle.Light),
				container.BorderTitle("Peers:"),
				container.PlaceWidget(peersList),
			),
			container.SplitFixed(3),
		),
	}

	debuggingLayout := []container.Option{
		container.SplitVertical(
			container.Left(peersLayout...),
			container.Right(
				container.PlaceWidget(errorMessages),
				container.Border(linestyle.Double),
				container.BorderColor(cell.ColorMaroon),
			),
			container.SplitPercent(30),
		),
	}

	return []container.Option{
		container.SplitHorizontal(
			container.Top(mainLayout...),
			container.Bottom(debuggingLayout...),
			container.SplitPercent(50),
		),
	}, nil
}

const (
	termboxTerminal = "termbox"
	tcellTerminal   = "tcell"
	rootID          = "root"
	redrawInterval  = 250 * time.Millisecond
)

var displayTerminal terminalapi.Terminal
var terminalCancel context.CancelFunc
var ctx context.Context

func setupDisplay() {
	terminalPtr := flag.String("terminal",
		"tcell",
		"The terminal implementation to use. Available implementations are 'termbox' and 'tcell' (default = tcell).")

	var err error
	switch terminal := *terminalPtr; terminal {
	case termboxTerminal:
		displayTerminal, err = termbox.New(termbox.ColorMode(terminalapi.ColorMode256))
	case tcellTerminal:
		displayTerminal, err = tcell.New(tcell.ColorMode(terminalapi.ColorMode256))
	default:
		log.Fatalf("Unknown terminal implementation '%s' specified. Please choose between 'termbox' and 'tcell'.", terminal)
		return
	}

	if err != nil {
		panic(err)
	}
	// init al the wigits
	textInput, err = messagingInput()
	if err != nil {
		panic(err)
	}
	messageText, err = newWrappedRollingText()
	if err != nil {
		panic(err)
	}
	peersList, err = newWrappedRollingText()
	if err != nil {
		panic(err)
	}

	connectField, err = ConnectInput()
	if err != nil {
		panic(err)
	}
	errorMessages, err = newWrappedRollingText()
	if err != nil {
		panic(err)
	}

	// place wigits
	layout, err := contLayout()
	if err != nil {
		panic(err)
	}
	c, err = container.New(displayTerminal, layout...)
	if err != nil {
		panic(err)
	}

	ctx, terminalCancel = context.WithCancel(context.Background())

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == keyboard.KeyEsc || k.Key == keyboard.KeyCtrlC {
			quit <- true
		}
	}
	go func() {
		if err := termdash.Run(ctx, displayTerminal, c, termdash.KeyboardSubscriber(quitter), termdash.RedrawInterval(redrawInterval)); err != nil {
			panic(err)
		}
	}()
}

func closeDisplay() {
	terminalCancel()
	displayTerminal.Close()
}
