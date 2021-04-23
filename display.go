package main

import (
	"context"
	"flag"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/text"
	"github.com/mum4k/termdash/widgets/textinput"
)

var textInput *textinput.TextInput
var messageText *text.Text
var errorMessages *text.Text
var peersList *text.Text
var connectField *textinput.TextInput

func messagingInput() (*textinput.TextInput, error) {
	input, err := textinput.New(
		textinput.Label("Message: ", cell.FgColor(cell.ColorSilver)),
		textinput.FillColor(cell.ColorDefault),
		textinput.Border(linestyle.Double),
		textinput.OnSubmit(func(text string) error {
			sendMessage(text)
			return nil
		}),
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
				enterNetwork("127.0.0.1:1234")
			} else {
				enterNetwork(text)
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

func WriteLn(t *text.Text, text string) {
	t.Write(text + "\n")
}

func displayPeers() {
	peersList.Reset()

	var ips []string
	for peer := range peers {
		ips = append(ips, peer.meta.GID+" "+strconv.Itoa(peer.meta.ConnectionCount))
	}
	sort.Strings(ips)

	for _, ip := range ips {
		WriteLn(peersList, ip)
	}
}

func contLayout() ([]container.Option, error) {
	messagingLayout := []container.Option{
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(textInput),
				container.PaddingRight(10),
				container.PaddingLeft(2),
			),
			container.Bottom(
				container.Border(linestyle.Light),
				container.BorderTitle("Chat:"),
				container.PlaceWidget(messageText),
			),
			container.SplitFixed(3),
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
			container.SplitFixed(2),
		),
	}

	return []container.Option{
		container.SplitHorizontal(
			container.Top(
				container.SplitVertical(
					container.Left(peersLayout...),
					container.Right(messagingLayout...),
					container.SplitPercent(30),
				),
			),
			container.Bottom(
				container.PlaceWidget(errorMessages),
				container.Border(linestyle.Double),
				container.BorderColor(cell.ColorMaroon),
			),
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
	c, err := container.New(displayTerminal, layout...)
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
