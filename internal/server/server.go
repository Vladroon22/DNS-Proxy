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
	cache         *cache.Cache
	buf_size      int
	isEnabledEDNS bool
	udpconn       *net.UDPConn
	udpaddr       *net.UDPAddr
	limit         *rate_limiter.Limiter
	// tcpaddr *net.TCPAddr
}

func DNSServer(udp *net.UDPAddr, rate int) *Server {
	return &Server{
		cache:         cache.InitCache(),
		udpaddr:       udp,
		limit:         rate_limiter.NewLimiter(rate),
		isEnabledEDNS: false,
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

	if s.isEnabledEDNS {
		s.buf_size = 4096
	} else {
		s.buf_size = 512
	}

	GoogleDNS := to_google.NewDNSReceiver(s.cache, s.buf_size, s.isEnabledEDNS)

	for {
		buffer := make([]byte, s.buf_size)
		b, remote, err := s.udpconn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
			continue
		}

		if isBanned, reason := s.limit.ProcessIP(string(remote.IP)); isBanned {
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

		questions, n, err := message.HandleQuestions(buffer[:b], header.Qdcount, s.cache)
		if err != nil {
			sendToClient([]byte(err.Error()), remote)
			continue
		}

		log.Println("cache ques:", n)
		if n == 0 {
			GoogleAnswer, err := GoogleDNS.RequestToGoogleDNS(buffer)
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
					respChan <- message.BuildResponse(&header, que, s.cache)
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
			header.Ancount = 1
			header.Qdcount = 1
			header.Arcount = 0
			header.Nscount = 0
			header.SetFlags(&header.Flags, 1, 0, 0, 0, 0, 0, 0, 0)
			response.Write(message.BuildResponse(&header, questions[0], s.cache))
		}

		if err := sendToClient(response.Bytes(), remote); err != nil {
			continue
		}
	}
}

func (s *Server) CloseUDP() error {
	return s.udpconn.Close()
}
