package server

import (
	"bytes"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/Vladroon22/DNS-Server/internal/cache"
	"github.com/Vladroon22/DNS-Server/internal/message"
	"github.com/Vladroon22/DNS-Server/internal/rate_limiter"
	"github.com/Vladroon22/DNS-Server/internal/to_google"
)

type Server struct {
	//	tcpconn *net.TCPListener
	buf_size int
	edns     bool
	udpconn  *net.UDPConn
	udpaddr  *net.UDPAddr
	limit    *rate_limiter.Limiter
	// tcpaddr *net.TCPAddr
}

func DNSServer(udp *net.UDPAddr, rate int) *Server {
	return &Server{
		udpaddr: udp,
		limit:   rate_limiter.NewLimiter(rate),
		edns:    false,
	}
}

func (s *Server) StartUDP() error {
	udp, err := net.ListenUDP("udp", s.udpaddr)
	if err != nil {
		return errors.New("failed to listen udp address: " + err.Error())
	}
	s.udpconn = udp

	go s.limit.StartLimiter()
	go s.acceptUDP()
	return nil
}

func (s *Server) acceptUDP() error {
	sendToClient := func(resp []byte, remote *net.UDPAddr) error {
		if _, err := s.udpconn.WriteToUDP(resp, remote); err != nil {
			log.Println(err)
			return err
		}
		return nil
	}

	caching := cache.InitCache()

	if s.edns {
		s.buf_size = 4096
	} else {
		s.buf_size = 512
	}

	for {
		buffer := make([]byte, s.buf_size)
		b, remote, err := s.udpconn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
			continue
		}

		isBanned, reason := s.limit.ProcessIP(string(remote.IP))
		if isBanned {
			sendToClient([]byte(reason), remote)
			continue
		}

		var response bytes.Buffer
		header, err := message.HandleHeader(buffer[:12])
		if err != nil {
			sendToClient([]byte(err.Error()), remote)
			continue
		}
		log.Println("Request header:", header)
		log.Println("questions:", header.Qdcount)

		questions, n, err := message.HandleQuestions(buffer[:b], header.Qdcount, caching)
		if err != nil {
			sendToClient([]byte(err.Error()), remote)
			continue
		}

		log.Println("cache ques:", n)
		if n == 0 {
			GoogleAnswer, err := to_google.RequestToGoogleDNS(buffer, caching, s.edns)
			if err != nil {
				sendToClient([]byte(err.Error()), remote)
				continue
			}
			log.Println("Google:", GoogleAnswer)
			response.Write(GoogleAnswer)
		} else if n > 1 {
			for _, que := range questions {
				log.Println("Cache question:", que)
			}

			header.SetFlags(&header.Flags, 1, 0, 0, 0, 0, 0, 0, 0)
			header.Qdcount = uint16(n)

			wg := sync.WaitGroup{}
			respChan := make(chan []byte, n)
			for _, que := range questions {
				wg.Add(1)
				go func(que message.Question) {
					defer wg.Done()
					respChan <- message.BuildResponse(&header, que, caching)
				}(que)
			}

			go func() {
				wg.Wait()
				close(respChan)
			}()

			for resp := range respChan {
				response.Write(resp)
			}
		} else if n == 1 {
			log.Println("Cache question:", questions)
			header.SetFlags(&header.Flags, 1, 0, 0, 0, 0, 0, 0, 0)
			header.Ancount = 1
			header.Qdcount = 1
			header.Arcount = 0
			header.Nscount = 0
			response.Write(message.BuildResponse(&header, questions[0], caching))
		}

		if err := sendToClient(response.Bytes(), remote); err != nil {
			continue
		}
	}
}

func (s *Server) CloseUDP() error {
	return s.udpconn.Close()
}

/*
	func (s *Server) StartTCP() error {
			tcp, err := net.ListenTCP("tcp", s.tcpaddr)
			if err != nil {
				return errors.New("failed to listen tcp address: " + err.Error())
			}
			s.tcpconn = tcp

			go s.acceptTCP()
			return nil
	}

	func (s *Server) acceptTCP() {
			sendToClient := func(conn *net.TCPConn, resp []byte) error {
				if _, err := conn.Write(resp); err != nil {
					log.Println(err)
					return err
				}
				return nil
			}

			for {
				buffer := make([]byte, 512)
				conn, err := s.tcpconn.AcceptTCP()
				if err != nil {
					log.Println(err)
					continue
				}

				header, err := message.HandleHeader(buffer[:12])
				if err != nil {
					sendToClient(conn, []byte(err.Error()))
					log.Println(err)
					continue
				}
				header.Decode()

				var response bytes.Buffer
				if err := sendToClient(conn, response.Bytes()); err != nil {
					continue
				}
			}
		}

		func (s *Server) CloseTCP() error {
			return s.tcpconn.Close()
		}
*/
