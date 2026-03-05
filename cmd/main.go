package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Vladroon22/DNS-Server/internal/logger"
	"github.com/Vladroon22/DNS-Server/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	myLogger := logger.NewLogger()
	go myLogger.StartLogging()
	defer myLogger.Stop()

	if err := godotenv.Load(); err != nil {
		myLogger.Log(logger.LogEntry{Info: err.Error()})
		os.Exit(1)
	}

	port, err := strconv.Atoi((os.Getenv("udp_port")))
	if err != nil {
		myLogger.Log(logger.LogEntry{Info: err.Error()})
		os.Exit(1)
	}

	stopServerChan := make(chan error, 1)
	stopOSChan := make(chan os.Signal, 1)
	signal.Notify(stopOSChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	configUDP := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port}

	srv := server.DNSServer(configUDP, 20, myLogger)
	myLogger.Log(logger.LogEntry{Info: fmt.Sprintf("Starting server on port %d", port)})

	go func() {
		if err := srv.StartUDP(); err != nil {
			stopServerChan <- err
		}
	}()

	select {
	case err := <-stopServerChan:
		myLogger.Log(logger.LogEntry{Info: fmt.Sprintf("DNS error: %v", err)})
		os.Exit(1)
	case sig := <-stopOSChan:
		myLogger.Log(logger.LogEntry{Info: fmt.Sprintf("Received %v signal, shutting down...", sig)})

		if err := srv.CloseUDP(context.Background()); err != nil {
			myLogger.Log(logger.LogEntry{Info: fmt.Sprintf("DNS shutdown error: %v", err)})
		}
	}
}
