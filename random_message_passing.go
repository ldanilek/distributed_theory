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

func NewMessageWithContent(from ProcessID, to ProcessID) Message {
	contents := fmt.Sprintf("%d", newContent())
	return MessageWithContent{
		Message: RoutedMessage{
			From: from,
			To: to,
		},
		Content: contents,
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
	send func(Message),
	receive func() Message,
) {
	switch rand.Intn(4) {
	case 1:
		m := NewMessageWithContent(p.Id(), p.PickNeighbor())
		send(m)
	case 2, 3:
		receive()
	}
}
