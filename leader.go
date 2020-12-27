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

func (p *LeaderElectionProcess) sendListToNeighbors(send func(RoutedMessage)) {
	for neighbor := range p.Neighbors {
		if neighbor == p.Id() { continue }
		send(RoutedMessage{
			Message: LeaderMessage{IdList: p.IdList},
			From: p.Id(),
			To: neighbor,
		})
	}
}

func (p *LeaderElectionProcess) receiveLeaderMessage(receive func() *RoutedMessage) bool {
	received := receive()
	if received == nil {
		return false
	}
	leaderMessage := received.Message.(LeaderMessage)
	// update state
	originalIDListLen := len(p.IdList)
	for id := range leaderMessage.IdList {
		p.IdList[id] = struct{}{}
	}
	return len(p.IdList) != originalIDListLen
}

func (p *LeaderElectionProcess) Step(send func(RoutedMessage), receive func() *RoutedMessage) {

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

type LeaderElectionCompleteScenario struct {
	GraphSize int
}

func (s LeaderElectionCompleteScenario) Network() Topology {
	processes := make([]Process, 0, s.GraphSize)
	allNodes := make(map[ProcessID] struct{})
	for i := 0; i < s.GraphSize; i++ {
		allNodes[ProcessID(i)] = struct{}{}
	}
	for i := 0; i < s.GraphSize; i++ {
		processes = append(processes, &LeaderElectionProcess{
			Process: &RandomProcess{
				IncrementalID: ProcessID(i),
				NeighborCount: s.GraphSize,
			},
			GraphSize: s.GraphSize,
			Neighbors: allNodes,
		})
	}
	return CompleteTopology(processes)
}

