RUN = docker run --rm -v $(CURDIR):/usr/src/distributed_theory -w /usr/src/distributed_theory golang:1.13-alpine

distributed_theory: lamport.go leader.go message.go network.go random_message_passing.go main.go sender_receiver.go tcp.go bellman_ford.go
	$(RUN) go build -v

run: distributed_theory
	$(RUN) ./distributed_theory
