package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type ProcessID int

func (id ProcessID) String() string {
	return fmt.Sprintf("pid:%d", int(id))
}

type Message interface {
	String() string
	Source() ProcessID
	Dest() ProcessID
}

type RoutedMessage struct {
	Message
	From ProcessID
	To ProcessID
}

func (m RoutedMessage) String() string {
	if m.Message != nil {
		return fmt.Sprintf("(%s) %s->%s", m.Message, m.From, m.To)
	}
	return fmt.Sprintf("%s->%s", m.From, m.To)
}

func (m RoutedMessage) Dest() ProcessID {
	return m.To
}

func (m RoutedMessage) Source() ProcessID {
	return m.From
}

type MessageWithContent struct {
	Message
	Content string
}

func (m MessageWithContent) String() string {
	if m.Message != nil {
		return fmt.Sprintf("\"%s\" %s", m.Content, m.Message)
	}
	return fmt.Sprintf("\"%s\"", m.Content)
}

// Generic processor. e.g. it can wrap a sub-algorithm with lamport clocks.
type Process interface {
	Id() ProcessID
	Step(
		send func(Message),
		receive func() Message,
	) (done bool)
}

type Cluster []ConnectedProcess

// A Process that knows how to communicate with other processes.
// It ignores the `send` and `receive`.
type DirectConnectedProcess struct {
	Process
	OutChans map[ProcessID]chan Message
	InChan chan Message
}
var _ ConnectedProcess = (*DirectConnectedProcess)(nil)

func (p *DirectConnectedProcess) ConnectedStep() {
	return p.Step(p.Send, p.Receive)
}

func (p *DirectConnectedProcess) Send(m Message) {
	// validate neighbor
	nbr := m.Dest()
	outChan, ok := p.OutChans[nbr]
	if !ok {
		panic(fmt.Sprintf("%d does not exist as a neighbor of %d", nbr, p.Id()))
	}
	select {
	case outChan <- m:
		// sent
	default:
		// dropped because direct channel is full
	}
}

func (p *DirectConnectedProcess) Receive() Message {
	select {
	case m := <-p.InChan:
		return m
	default:
		return nil
	}
}

func (c Cluster) RunRandomSteps() {
	var wg sync.WaitGroup
	wg.Add(len(c))
	for _, process := range c {
		process := process
		go func() {
			defer wg.Done()
			process.RunRandomSteps()
		}()
	}
	wg.Wait()
}

func (c Cluster) CreateProcessConnectedToOthers() *Process {
	outChans := make(map[ProcessID]chan Message, len(c))
	for _, p := range c {
		outChans[p.Id] = p.InChan
	}
	inChan := make(chan Message, 1000)
	newProcess := &Process{
		Id: ProcessID(len(c)),
		OutChans: outChans,
		InChan: inChan,
	}
	for _, p := range c {
		p.OutChans[newProcess.Id] = inChan
	}
	return newProcess
}

func CreateProcesses(count int) Cluster {
	c := Cluster(make([]*Process, 0, count))
	for i := 0; i<count; i++ {
		p := c.CreateProcessConnectedToOthers()
		c = append(c, p)
	}
	return c
}

