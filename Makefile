.PHOHY:

run:
	go run cmd/main.go

docker: 
	sudo docker build -t mydns .
	sudo docker run --name=custom-dns -p 8530:8530 -d mydns