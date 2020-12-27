package main 

func main() {
	switch 2 {
	case 0: 
		RunScenario(RandomWithLamportScenario{
			NumProcs: 10,
		})
	case 1:
		RunScenario(LeaderElectionCompleteScenario{
			GraphSize: 10,
		})
	case 2:
		RunScenario(BellmanFordScenario{})
	}
}
