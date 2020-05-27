package main

import (
	"fmt"
)
// Defines a Process that performs a leader election before deferring to its wrapped process.


type LeaderElectionProcess struct {
	Process
	GraphSize int
	LeaderId ProcessID
	LeaderFound bool
	IdList map[ProcessID] struct{}
	Neighbors map[ProcessID]struct{}
}

type LeaderMessage struct {
	Message
	IdList map[ProcessID] struct{}
}

func (m LeaderMessage) String() string {
	return fmt.Sprintf("Message: %s | IdList: %v", m.Message, m.IdList)
}

func (p *LeaderElectionProcess) sendListToNeighbors(send func(Message)) {
	for neighbor := range p.Neighbors {
		if neighbor == p.Id() { continue }
		routed_msg := RoutedMessage {
			From : p.Id(),
			To: neighbor,
		}
		send(LeaderMessage {Message : routed_msg, IdList: p.IdList})
	}
}

func (p *LeaderElectionProcess) receiveLeaderMessage(receive func() Message) bool {
	var received_msg Message
	for received_msg == nil {
		received_msg = receive()
	}
	leader_msg := received_msg.(LeaderMessage)
	// update state
	len_id_orig := len(p.IdList)
	for id := range leader_msg.IdList {
		p.IdList[id] = struct{}{}
	}
	return len(p.IdList) != len_id_orig
}


func (p *LeaderElectionProcess) Step(send func(Message), receive func() Message) {

	if p.LeaderFound {
		return
	}
	// init 
	p.LeaderId = p.Id()
	p.IdList = make(map[ProcessID] struct{})
	p.IdList[p.LeaderId] = struct{}{}

	// main loop
	p.sendListToNeighbors(send)
	for {
		updated := p.receiveLeaderMessage(receive)
		if updated {
			p.sendListToNeighbors(send)
			if len(p.IdList) == p.GraphSize	{
				p.LeaderFound = true
				break
			}
		}
	}
	for id := range p.IdList {
		if id > p.LeaderId {
			p.LeaderId = id 
		}
	}
	Log(p, fmt.Sprintf("found leader %s", p.LeaderId))
}

func RunLeaderElectionCompleteGraph (graphsize int) {
	processes := make([]Process, 0, graphsize)
	allNodes := make(map[ProcessID] struct{})
	for i := 0; i < graphsize; i++ {
		allNodes[ProcessID(i)] = struct{}{}
	}
	for i := 0; i < graphsize; i++ {
		processes = append(processes, &LeaderElectionProcess{
			Process: &RandomProcess{
				IncrementalID: ProcessID(i),
				NeighborCount: graphsize,
			},
			GraphSize: graphsize,
			Neighbors: allNodes,
		})
	}
	c := CreateCompleteGraph(processes)
	c.RunTillDone()
}


