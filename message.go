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
}

type BaseMessage struct {
	From ProcessID
	To ProcessID
}

func (m BaseMessage) String() string {
	if m.From == m.To && m.From == 0 {
		return ""
	}
	return fmt.Sprintf("%s->%s", m.From, m.To)
}

func BaseMessageFactory(from ProcessID, to ProcessID) Message {
	return BaseMessage{From: from, To: to}
}

type MessageWithContent struct {
	Message
	Content string
}

func (m MessageWithContent) String() string {
	return fmt.Sprintf("\"%s\" %s", m.Content, m.Message)
}

func MessageWithContentFactory(from ProcessID, to ProcessID) Message {
	contents := fmt.Sprintf("%d", rand.Intn(1000))
	return MessageWithContent{
		Message: BaseMessageFactory(from, to),
		Content: contents,
	}
}

type Event struct {
	EventType string
	Message Message
}

type Network interface {
}

type Process interface {
	Send(nbr ProcessID, m Message)
	Receive()
	Compute()
}

type Cluster []Process

type BaseProcess struct {
	Id ProcessID
	Log []Event
	OutChans map[ProcessID]chan Message
	InChan chan Message
}

func (p *BaseProcess) Send(nbr ProcessID, m Message) {
	// validate neighbor
	outChan, ok := p.OutChans[nbr]
	if !ok {
		panic(fmt.Sprintf("%d does not exist as a neighbor of %d", nbr, p.Id))
	}
	p.Log = append(p.Log, Event{
		EventType: "sent",
		Message: m,
	})
	outChan <- m
}

// reads everything available from InChan
func (p *BaseProcess) Receive() {
	select {
	case m := <-p.InChan:
		p.Log = append(p.Log, Event{
			EventType: "received",
			Message: m,
		})
	default:
		return
	}
}

func (p *BaseProcess) Compute() {
	p.Log = append(p.Log, Event{EventType: "S"})
}

func (p *BaseProcess) PickNeighbor() ProcessID {
	nbr := p.Id
	for nbr == p.Id {
		nbr = ProcessID(rand.Intn(len(p.OutChans)))
	}
}

func RunRandomStep(p Process) {
	switch rand.Intn(4) {
	case 0:
		p.Compute()
	case 1:
		p.Send(nbr)
	case 2, 3:
		p.Receive()
	}
}

const MaxSteps = 100

func RunRandomSteps(p Process) {
	steps := rand.Intn(MaxSteps)
	for i := 0; i < steps; i++ {
		p.RunRandomStep()
		time.Sleep(1)  // yield
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

