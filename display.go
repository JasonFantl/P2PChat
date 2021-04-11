// Copyright 2019 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Binary textinputdemo shows the functionality of a text input field.
package main

import (
	"context"
	"flag"
	"log"
	"time"

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
var connectButton *button.Button

func newTextInput() (*textinput.TextInput, error) {
	input, err := textinput.New(
		textinput.Label("Message: ", cell.FgColor(cell.ColorNumber(33))),
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

func contLayout() ([]container.Option, error) {
	messagingLayout := []container.Option{
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(textInput),
			),
			container.Bottom(
				container.Border(linestyle.Light),
				container.BorderTitle("Chat:"),
				container.PlaceWidget(messageText),
			),
			container.SplitFixed(2),
		),
	}

	peersLayout := []container.Option{
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(connectButton),
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
			),
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
	flag.Parse()

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
	textInput, err = newTextInput()
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

	connectButton, err = button.New("CONNECT", func() error {
		connectToPeer("127.0.0.1", "1234")
		return nil
	},
		button.FillColor(cell.ColorNumber(196)),
		button.Height(1),
	)
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
			close()
		}
	}
	go func() {
		if err := termdash.Run(ctx, displayTerminal, c, termdash.KeyboardSubscriber(quitter), termdash.RedrawInterval(redrawInterval)); err != nil {
			panic(err)
		}
	}()
}
