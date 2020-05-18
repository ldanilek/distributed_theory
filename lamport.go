package main

import (
	"fmt"
	"math/rand"
	"sync"
)

type ProcessID int

type Message struct {
	From ProcessID
	To ProcessID
	Contents string
}

type Event struct {
	EventType string
	Message Message  // optional
}

type Process struct {
	Id ProcessID
	Log []Event
	OutChans map[ProcessID]chan Message
	InChan chan Message
}
type Cluster []*Process

func (p *Process) Send(nbr ProcessID, contents string) {
	// validate neighbor
	outChan, ok := p.OutChans[nbr]
	if !ok {
		panic(fmt.Sprintf("%d does not exist as a neighbor of %d", nbr, p.Id))
	}
	m := Message{
		From: p.Id,
		To: nbr,
		Contents: contents,
	}
	p.Log = append(p.Log, Event{
		EventType: "sent message",
		Message: m,
	})
	outChan <- m
}

// reads everything available from InChan
func (p *Process) Receive() {
	for {
		select {
		case m := <-p.InChan:
			p.Log = append(p.Log, Event{
				EventType: "received message",
				Message: m,
			})
		default:
			return
		}
	}
}

func (p *Process) RunRandomStep() {
	switch rand.Intn(3) {
	case 0:
		p.Log = append(p.Log, Event{EventType: "state change"})
	case 1:
		nbr := ProcessID(rand.Intn(len(p.OutChans)))
		contents := fmt.Sprintf("message %d", rand.Intn(1000))
		p.Send(nbr, contents)
	case 2:
		p.Receive()
	}
}

func (p *Process) RunRandomSteps() {
	for i := 0; i < rand.Intn(10); i++ {
		p.RunRandomStep()
	}
	p.Receive()
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

func main() {
	c := CreateProcesses(10)
	c.RunRandomSteps()
	for _, p := range c {
		fmt.Println(p.Log)
	}
}
