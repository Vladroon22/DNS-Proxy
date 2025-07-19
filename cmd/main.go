package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Vladroon22/DNS-Server/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalln(err)
	}

	portUDP, errUDP := strconv.Atoi((os.Getenv("udp_port")))
	if errUDP != nil {
		log.Fatalln(errUDP)
	}

	configUDP := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: portUDP}

	srv := server.DNSServer(configUDP, 20)
	log.Printf("DNS is running on udp: %d\n", 8536)

	go func() {
		if err := srv.StartUDP(); err != nil {
			log.Println(err)
			return
		}
	}()

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, syscall.SIGINT, syscall.SIGTERM)
	<-exitCh

	go func() {
		if err := srv.CloseUDP(); err != nil {
			log.Println(err)
			return
		}
	}()
}
