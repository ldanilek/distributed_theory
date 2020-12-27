package main

import (
	"fmt"
)

const (
	patience = 100
)

type ConversationProcess struct{
	ID ProcessID
	FriendID ProcessID
	ExpectedPhraseIndex int
	Phrases []string
	Initiate bool
	PhraseIndex int
}

func (p *ConversationProcess) Id() ProcessID {
	return p.ID
}

type ConversationMessage struct {
	Phrase string
}

func (m ConversationMessage) String() string {
	return m.Phrase
}

func (p *ConversationProcess) sendMessage(send func(RoutedMessage)) {
	if p.PhraseIndex >= len(p.Phrases) {
		return
	}
	send(RoutedMessage{
		Message: ConversationMessage{
			Phrase: p.Phrases[p.PhraseIndex],
		},
		From: p.Id(),
		To: p.FriendID,
	})
	p.PhraseIndex += 1
}

func (p *ConversationProcess) Step(send func(RoutedMessage), receive func() *RoutedMessage) {
	if p.Initiate {
		p.sendMessage(send)
		p.Initiate = false
	}
	received := receive()
	if received == nil {
		return
	}
	if received.From != p.FriendID {
		panic(fmt.Sprintf("unexpected source of conversation: %v", received.From))
	}
	contentMessage, ok := received.Message.(ConversationMessage)
	if !ok {
		panic(fmt.Sprintf("conversation expects phrase, not %T %s", received.Message, received.Message))
	}
	Log(p, fmt.Sprintf("received phrase '%s'", contentMessage))
	p.sendMessage(send)
	if p.PhraseIndex >= len(p.Phrases) {
		Log(p, "conversation complete")
	}
}

