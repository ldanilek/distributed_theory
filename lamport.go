package main

import (
	"fmt"
	"math/rand"
)

type Message struct {
	Contents int
}

type Process struct {
	Id int
	State int
	OutChans map[int]<-chan Message
	InChans map[int]chan-> Message
}

func (p *Process) Send(nbr int, m Message) {
	p.OutChan[nbr] <- m
}

func CreateProcessConnectedToOthers(processes []*Process) *Process {
	maxProcessID := 0
	outChans := make(map[int]<-chan )
	for _, p := range processes {
		if p.Id > maxProcessIDs {
			maxProcessID = p.Id
		}
	}
	newProcess := &Process{
		Id: maxProcessID + 1,
		State: 0,
		OutChans
	}
}

func main() {
	fmt.Println("HI")
}
