package main

import (
	"fmt"
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
		return fmt.Sprintf("\"%s\"\t%s", m.Content, m.Message)
	}
	return fmt.Sprintf("\"%s\"", m.Content)
}

// Generic processor.
// Examples:
// It can send random messages to its neighbors.
// It can wrap a sub-algorithm with lamport clocks.
type Process interface {
	Id() ProcessID
	Step(
		send func(Message),
		receive func() Message,
	)
}

func Log(p Process, s string) {
	fmt.Printf("%s: %s\n", p.Id(), s)
}

// A Process that knows how to communicate with other processes.
// It knows how to `send` and `receive`.
// It should always be the topmost process, so it is technically not a Process,
// to avoid embedding it and causing confusion.
type DirectConnectedProcess struct {
	P Process
	OutChans map[ProcessID]chan Message
	InChan chan Message
}

func (p *DirectConnectedProcess) ConnectedStep() {
	p.P.Step(p.Send, p.Receive)
}

// Actually infinite loops
// TODO: track whether channels are empty, and ask process if it's idle.
func (p *DirectConnectedProcess) RunTillDone() {
	for {
		p.ConnectedStep() 
		time.Sleep(1) // yield
	}
	Log(p.P, "done")
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
		Log(p.P, fmt.Sprintf("sent %s", m))
		// sent
	default:
		// dropped because direct channel is full
	}
}

func (p *DirectConnectedProcess) Receive() Message {
	select {
	case m := <-p.InChan:
		Log(p.P, fmt.Sprintf("received %s", m))
		return m
	default:
		return nil
	}
}

func (p *DirectConnectedProcess) Id() ProcessID {
	return p.P.Id()
}

type Cluster map[ProcessID]*DirectConnectedProcess

func (c Cluster) RunTillDone() {
	var wg sync.WaitGroup
	wg.Add(len(c))
	for _, process := range c {
		process := process
		go func() {
			defer wg.Done()
			process.RunTillDone()
		}()
	}
	wg.Wait()
}

type TopologyNode struct{
	Subprocess Process
	Neighbors map[ProcessID]struct{}
}

type Topology map[ProcessID]TopologyNode

func CreateCluster(topo Topology) Cluster {
	c := make(Cluster, len(topo))
	for id, topoNode := range topo {
		if id != topoNode.Subprocess.Id() {
			panic(fmt.Sprintf("invalid topology: %s has subprocess %s", id, topoNode.Subprocess.Id()))
		}
		inChan := make(chan Message, 1000)
		c[id] = &DirectConnectedProcess{
			P: topoNode.Subprocess,
			InChan: inChan,
		}
	}
	for id, topoNode := range topo {
		outChans := make(map[ProcessID]chan Message, len(topoNode.Neighbors))
		for pid := range topoNode.Neighbors {
			if id == pid {
				// Skip links to self.
				continue
			}
			outChans[pid] = c[pid].InChan
		}
		c[id].OutChans = outChans
	}
	return c
}

func CreateCompleteGraph(processes []Process) Cluster {
	allPIDs := make(map[ProcessID]struct{}, len(processes))
	for _, p := range processes {
		allPIDs[p.Id()] = struct{}{}
	}
	topo := make(Topology, len(processes))
	for _, p := range processes {
		topo[p.Id()] = TopologyNode{
			Subprocess: p,
			Neighbors: allPIDs,
		}
	}
	return CreateCluster(topo)
}

