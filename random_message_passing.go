package distributed_theory

import (
	"fmt"
	"math/rand"
)

func MessageWithRandomContent(from ProcessID, to ProcessID) Message {
	contents := fmt.Sprintf("%d", rand.Intn(1000))
	return MessageWithContent{
		Message: RoutedMessage{
			From: from,
			To: to,
		},
		Content: contents,
	}
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
