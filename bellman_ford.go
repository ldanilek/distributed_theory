package main

import (
	"fmt"
)

type NextStep struct {
	steppingStone ProcessID
	totalDistance int
}

type BellmanFordProcess struct {
	Process
	shortestNextSteps map[ProcessID]NextStep
	hasStarted bool
	neighbors map[ProcessID]struct{}
	deliveryQueue chan RoutedMessage
}

const (
	innerDeliveryQueueSize = 1000
)

func NewBellmanFordTopologyNode(
	subprocess Process,
	neighbors map[ProcessID]struct{},
) TopologyNode {
	return TopologyNode{
		Subprocess: &MultiTCPProcess{
			Process: &BellmanFordProcess{
				Process: subprocess,
				shortestNextSteps: map[ProcessID]NextStep{
					subprocess.Id(): NextStep{steppingStone: subprocess.Id()},
				},
				neighbors: neighbors,
				deliveryQueue: make(chan RoutedMessage, innerDeliveryQueueSize),
			},
		},
		Neighbors: neighbors,
	}
}


func (p *BellmanFordProcess) broadcast(send func(RoutedMessage)) {
	for neighbor := range p.neighbors {
		send(RoutedMessage{
			Message: BellmanFordUpdateMessage{
				shortestNextSteps: p.shortestNextSteps,
			},
			From: p.Id(),
			To: neighbor,
		})
	}
}

func (p *BellmanFordProcess) update(
	m BellmanFordUpdateMessage, 
	from ProcessID,
) bool {
	hadUpdate := false
	for dest, nextStep := range m.shortestNextSteps {
		if knownNextStep, ok := p.shortestNextSteps[dest]; ok {
			if knownNextStep.totalDistance > 1 + nextStep.totalDistance {
				p.shortestNextSteps[dest] = NextStep{
					steppingStone: from,
					totalDistance: 1 + nextStep.totalDistance,
				}
				hadUpdate = true
			}	
		} else {
			p.shortestNextSteps[dest] = NextStep{
				steppingStone: from,
				totalDistance: 1 + nextStep.totalDistance,
			}
			hadUpdate = true
		}
	}
	return hadUpdate
}

func (p *BellmanFordProcess) innerStep(send func(RoutedMessage)) {
	if p.Process == nil {
		return
	}
	p.Process.Step(
		func(m RoutedMessage) {
			if nextStep, ok := p.shortestNextSteps[m.To]; ok {
				send(RoutedMessage{
					Message: m,
					From: p.Id(),
					To: nextStep.steppingStone,
				})
			} else {
				Log(p, fmt.Sprintf("unable to deliver message %v", m))
			}
		},
		func() *RoutedMessage {
			select {
			case received := <-p.deliveryQueue:
				return &received
			default:
				return nil
			}
		},
	)
}


func (p *BellmanFordProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	defer p.innerStep(send)
	if !p.hasStarted {
		p.broadcast(send)
		p.hasStarted = true
	}
	// We need to look for messages even if our 
	// inner process doesn't call 'receive',
	// to receive Update messages as they arrive. 
	// If we get a message that should be passed
	// to the inner process, stick it in p.deliveryQueue.
	received := receive()
	if received == nil {
		return
	}
	messageReceived := received.Message
	if updateMessage, ok := messageReceived.(BellmanFordUpdateMessage); ok {
		if p.update(updateMessage, received.From) {
			p.broadcast(send)
		}
	} else if innerMessage, ok := messageReceived.(RoutedMessage); ok {
		if innerMessage.To == p.Id() {
			select {
			case p.deliveryQueue <- innerMessage:
				// delivered
			default:
				// queue full; drop the message
			}
		} else {
			// pass it on, if we know the route
			if nextStep, ok := p.shortestNextSteps[innerMessage.To]; ok {
				send(RoutedMessage{
					Message: innerMessage,
					From: p.Id(),
					To: nextStep.steppingStone,
				})
			} else {
				// Otherwise, drop it.
				Log(p, fmt.Sprintf("unable to deliver message %v", innerMessage))
			}
		}
	} else {
		panic(fmt.Sprintf(
			"bellman ford expected either " +
			"BellmanFordUpdateMessage or RoutedMessage",
		))
	}
}

type BellmanFordUpdateMessage struct {
	shortestNextSteps map[ProcessID]NextStep
}

func (m BellmanFordUpdateMessage) String() string {
	return fmt.Sprintf("next steps: %v", m.shortestNextSteps)
}

type BellmanFordScenario struct{}

func (s BellmanFordScenario) Network() Topology {
	//     2  -  3
	//   /         \
	// 1 - 4 - 5 - 6 - 8
	//   \     |     /
	//     -   7   - 
	// node 1 tries to communicate with 8, who sends an ACK
	return map[ProcessID]TopologyNode {
		1: NewBellmanFordTopologyNode(
			&ConversationProcess{
				ID: 1,
				FriendID: 8,
				Phrases: []string{"hi there", "what's up"},
				Initiate: true,
			},
			map[ProcessID]struct{}{2: {}, 4: {}, 7: {}},
		),
		2: NewBellmanFordTopologyNode(SimpleProcess{ID: 2},
			map[ProcessID]struct{}{1: {}, 3: {}},
		),
		3: NewBellmanFordTopologyNode(SimpleProcess{ID: 3},
			map[ProcessID]struct{}{2: {}, 6: {}},
		),
		4: NewBellmanFordTopologyNode(SimpleProcess{ID: 4},
			map[ProcessID]struct{}{1: {}, 5: {}},
		),
		5: NewBellmanFordTopologyNode(SimpleProcess{ID: 5},
			map[ProcessID]struct{}{4: {}, 6: {}, 7: {}},
		),
		6: NewBellmanFordTopologyNode(SimpleProcess{ID: 6},
			map[ProcessID]struct{}{3: {}, 5: {}, 8: {}},
		),
		7: NewBellmanFordTopologyNode(SimpleProcess{ID: 7},
			map[ProcessID]struct{}{1: {}, 5: {}, 8: {}},
		),
		8: NewBellmanFordTopologyNode(
			&ConversationProcess{
				ID: 8,
				FriendID: 1,
				Phrases: []string{"hi", "all good"},
			},
			map[ProcessID]struct{}{6: {}, 7: {}},
		),
	}
}
