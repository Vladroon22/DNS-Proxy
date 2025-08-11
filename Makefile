.PHOHY:

run:
	go run cmd/main.go

check-race:
	go run ./cmd/main.go --race

docker: 
	sudo docker build -t mydns .
	sudo docker run --name=custom-dns -p 8530:8530 -d mydns