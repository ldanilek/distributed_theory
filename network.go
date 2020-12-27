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

// When wrapping, the inner message contains "data" or "contents", while the
// outer message adds "metadata".
// For example, a message containing random data might be wrapped inside
// one with a lamport clock, which would be inside a RoutedMessage.
type Message interface {
	String() string
}

// Always the outermost wrapping of a message, so it can be sent over the wire.
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
// When wrapping, the inner process does more business logic,
// while the outer does more bookkeeping.
type Process interface {
	Id() ProcessID
	Step(
		send func(RoutedMessage),
		receive func() *RoutedMessage,
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
	OutChans map[ProcessID]chan RoutedMessage
	InChan chan RoutedMessage
}

func (p *DirectConnectedProcess) Step() {
	p.P.Step(
		func(m RoutedMessage) {
			p.Send(m)
		},
		func() *RoutedMessage {
			return p.Receive()
		},
	)
}

// Actually infinite loops
// TODO: track whether channels are empty, and ask process if it's idle.
func (p *DirectConnectedProcess) RunTillDone() {
	for {
		p.Step() 
		time.Sleep(10 * time.Millisecond) // yield
	}
	Log(p.P, "done")
}

func (p *DirectConnectedProcess) Send(m RoutedMessage) {
	if m.From != p.Id() {
		panic(fmt.Sprintf("%s cannot send a message that is 'from' %s", p.Id(), m.From))
	}
	// validate neighbor
	nbr := m.To
	outChan, ok := p.OutChans[nbr]
	if !ok {
		panic(fmt.Sprintf("%d does not exist as a neighbor of %d", nbr, p.Id()))
	}
	select {
	case outChan <- m:
		// Log(p.P, fmt.Sprintf("sent %s", m))
		// sent
	default:
		// dropped because direct channel is full
	}
}

func (p *DirectConnectedProcess) Receive() *RoutedMessage {
	select {
	case m := <-p.InChan:
		// Log(p.P, fmt.Sprintf("received %s", m))
		return &m
	default:
		return nil
	}
}

func (p *DirectConnectedProcess) Id() ProcessID {
	return p.P.Id()
}

type SimpleProcess struct {
	ID ProcessID
}

func (p SimpleProcess) Id() ProcessID {
	return p.ID
}

func (p SimpleProcess) Step(send func(RoutedMessage), receive func() *RoutedMessage) {
	received := receive()
	if received != nil {
		panic(fmt.Sprintf("simple process can't receive anything. received %T %v", received, received))
	}
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

const (
	directChannelBuffer = 1000
)

func CreateCluster(topo Topology) Cluster {
	c := make(Cluster, len(topo))
	for id, topoNode := range topo {
		if id != topoNode.Subprocess.Id() {
			panic(fmt.Sprintf("invalid topology: %s has subprocess %s", id, topoNode.Subprocess.Id()))
		}
		inChan := make(chan RoutedMessage, directChannelBuffer)
		c[id] = &DirectConnectedProcess{
			P: topoNode.Subprocess,
			InChan: inChan,
		}
	}
	for id, topoNode := range topo {
		outChans := make(map[ProcessID]chan RoutedMessage, len(topoNode.Neighbors))
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

func CompleteTopology(processes []Process) Topology {
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
	return topo
}

func CreateCompleteGraph(processes []Process) Cluster {
	return CreateCluster(CompleteTopology(processes))
}

type Scenario interface {
	Network() Topology
}

func RunScenario(scenario Scenario) {
	CreateCluster(scenario.Network()).RunTillDone()
}
