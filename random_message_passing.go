package main

import (
	"fmt"
	"math/rand"
	"sync"
)

var (
	incrementingContent int
	incrementingContentMutex sync.Mutex
)

func newContent() int {
	incrementingContentMutex.Lock()
	defer incrementingContentMutex.Unlock()
	incrementingContent++
	return incrementingContent
}

func NewMessageWithContent(from ProcessID, to ProcessID) RoutedMessage {
	contents := fmt.Sprintf("%d", newContent())
	return RoutedMessage{
		Message: MessageWithContent{
			Content: contents,
		},
		From: from,
		To: to,
	}
}

type RandomProcess struct {
	IncrementalID ProcessID
	// Neighbors have ids [0, NeighborCount)
	NeighborCount int
}

func (p *RandomProcess) PickNeighbor() ProcessID {
	nbr := p.Id()
	for nbr == p.Id() {
		nbr = ProcessID(rand.Intn(p.NeighborCount))
	}
	return nbr
}

func (p *RandomProcess) Id() ProcessID {
	return p.IncrementalID
}

func (p *RandomProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	switch rand.Intn(4) {
	case 1:
		m := NewMessageWithContent(p.Id(), p.PickNeighbor())
		send(m)
	case 2, 3:
		receive()
	}
}
