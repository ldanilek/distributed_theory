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

func (p *LamportProcess) Step(
	send func(Message),
	receive func() Message,
) (done bool) {
	return p.Process.Step(
		func(m Message) {
			send(LamportMessage{Message: m, Clock: p.Clock})
		},
		func() Message {
			m := receive()
			clock := p.Clock
			if clock < m.Clock {
				clock = m.Clock
			}
			p.Clock = clock + 1
		},
	)
}

func main() {
	c := CreateProcesses(10)
	c.RunRandomSteps()
	for _, p := range c {
		fmt.Println(p.Id)
		fmt.Println(p.Log)
	}
}
