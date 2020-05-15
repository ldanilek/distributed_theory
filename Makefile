RUN = docker run --rm -v $(CURDIR):/usr/src/distributed_theory -w /usr/src/distributed_theory golang:1.13-alpine

lamport: lamport.go
	$(RUN) go build -v

run: lamport
	$(RUN) ./distributed_theory
