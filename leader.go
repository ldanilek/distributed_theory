package main

// Defines a Process that performs a leader election before deferring to its wrapped process.


type LeaderElectionProcess struct {
	Process
	Neighbors map[ProcessID]struct{}
}
