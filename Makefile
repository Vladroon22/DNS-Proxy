.PHOHY:

run:
	go build cmd/main.go
	./main

check-race:
	go run ./cmd/main.go --race