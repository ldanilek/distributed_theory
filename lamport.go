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
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	p.Process.Step(
		func(m RoutedMessage) {
			send(RoutedMessage{
				Message: LamportMessage{Message: m.Message, Clock: p.Clock},
				From: m.From,
				To: m.To,
			})
			p.Clock++
		},
		func() *RoutedMessage {
			mRaw := receive()
			if mRaw == nil {
				return nil
			}
			m, ok := mRaw.Message.(LamportMessage)
			if !ok {
				panic(fmt.Sprintf("unexpected lamport type %T", mRaw.Message))
			}
			clock := p.Clock
			if clock < m.Clock {
				clock = m.Clock
			}
			p.Clock = clock + 1
			return &RoutedMessage{
				Message: m.Message,
				From: mRaw.From,
				To: mRaw.To,
			}
		},
	)
}

type RandomWithLamportScenario struct {
	NumProcs int
}

func (s RandomWithLamportScenario) Network() Topology {
	processes := make([]Process, 0, s.NumProcs)
	for i := 0; i < s.NumProcs; i++ {
		processes = append(processes, &LamportProcess{
			Process: &RandomProcess{
				IncrementalID: ProcessID(i),
				NeighborCount: s.NumProcs,
			},
			Clock: 0,
		})
	}
	return CompleteTopology(processes)
}

