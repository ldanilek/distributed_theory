package main

import (
	"fmt"
)

type LamportProcess struct {
	Process
	Clock int
}

type LamportMessage struct {
	Message
	Clock int
}

func (m LamportMessage) String() string {
	return fmt.Sprintf("%s (%d)", m.Message, m.Clock)
}

func (p *LamportProcess) Step(
	send func(Message),
	receive func() Message,
) {
	p.Process.Step(
		func(m Message) {
			send(LamportMessage{Message: m, Clock: p.Clock})
			p.Clock++
		},
		func() Message {
			mRaw := receive()
			if mRaw == nil {
				return nil
			}
			m, ok := mRaw.(LamportMessage)
			if !ok {
				fmt.Printf("dropping %s\n", mRaw)
			}
			clock := p.Clock
			if clock < m.Clock {
				clock = m.Clock
			}
			p.Clock = clock + 1
			return m.Message
		},
	)
}

const NumProcs = 10

func RunRandomWithLamportClocks() {
	processes := make([]Process, 0, NumProcs)
	for i := 0; i < NumProcs; i++ {
		processes = append(processes, &LamportProcess{
			Process: &RandomProcess{
				IncrementalID: ProcessID(i),
				NeighborCount: NumProcs,
			},
			Clock: 0,
		})
	}
	c := CreateCompleteGraph(processes)
	c.RunTillDone()
}

func main() {
	// RunRandomWithLamportClocks()
	RunLeaderElectionCompleteGraph(NumProcs)
}
