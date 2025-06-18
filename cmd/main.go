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

	rps, err := strconv.Atoi(os.Getenv("rps"))
	if err != nil {
		log.Fatalln(err)
	}

	/*
		portTCP, errTcp := strconv.Atoi((os.Getenv("tcp_port")))
		if errTcp != nil {
			portTCP = 5400
		}
	*/
	configUDP := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: portUDP}
	//configTCP := &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: portTCP}

	srv := server.DNSServer(configUDP, rps)
	log.Printf("DNS is running on udp: %d\n", portUDP)

	go func() {
		if err := srv.StartUDP(); err != nil {
			log.Println(err)
			return
		}
	}()

	//go func() {
	//	if err := srv.StartTCP(); err != nil {
	//		log.Println(err)
	//		return
	//	}
	//}()

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, syscall.SIGINT, syscall.SIGTERM)
	<-exitCh

	go func() {
		//	if err := srv.CloseTCP(); err != nil {
		//		log.Println(err)
		//		return
		//	}
		if err := srv.CloseUDP(); err != nil {
			log.Println(err)
			return
		}
	}()
}
