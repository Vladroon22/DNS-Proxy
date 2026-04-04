package server

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Vladroon22/DNS-Server/internal/cache"
	"github.com/Vladroon22/DNS-Server/internal/logger"
	"github.com/Vladroon22/DNS-Server/internal/message"
	"github.com/Vladroon22/DNS-Server/internal/rate_limiter"
	"github.com/Vladroon22/DNS-Server/internal/to_google"
)

type Server struct {
	cache         *cache.Cache
	bufSize       int
	isEnabledEDNS bool
	udpConn       *net.UDPConn
	udpAddr       *net.UDPAddr
	limit         *rate_limiter.Limiter
	logger        *logger.Logger
}

func DNSServer(udp *net.UDPAddr, rate int, lg *logger.Logger) *Server {
	return &Server{
		cache:         cache.InitCache(),
		udpAddr:       udp,
		limit:         rate_limiter.NewLimiter(rate),
		isEnabledEDNS: false,
		logger:        lg,
	}
}

func (s *Server) StartUDP() error {
	udp, err := net.ListenUDP("udp", s.udpAddr)
	if err != nil {
		return err
	}
	s.udpConn = udp

	if s.isEnabledEDNS {
		s.bufSize = 4096
	} else {
		s.bufSize = 512
	}

	go s.limit.StartLimiter()
	go s.acceptUDP()

	return nil
}

func (s *Server) sendToClient(resp []byte, remote *net.UDPAddr) error {
	if _, err := s.udpConn.WriteToUDP(resp, remote); err != nil {
		s.logger.Log(logger.LogEntry{Info: err.Error()})
		return err
	}
	return nil
}

func (s *Server) acceptUDP() {
	GoogleDNS := to_google.NewDNSReceiver(s.cache, s.bufSize, s.isEnabledEDNS, s.logger)

	bufPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, s.bufSize)
		},
	}

	for {
		buf := bufPool.Get().([]byte)
		b, remote, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			s.logger.Log(logger.LogEntry{Info: err.Error()})
			return
		}

		go func(c context.Context, buffer []byte, n int, remote *net.UDPAddr) {
			ctx, cancel := context.WithTimeout(c, time.Second*30)
			defer func() {
				bufPool.Put(buffer)
				cancel()
			}()

			select {
			case <-ctx.Done():
				if err := s.sendToClient([]byte("Request denied due server's issues"), remote); err != nil {
					s.logger.Log(logger.LogEntry{Info: err.Error()})
					return
				}
				s.logger.Log(logger.LogEntry{Info: err.Error()})
				return
			default:

				if isBanned, reason := s.limit.ProcessIP(string(remote.IP)); isBanned {
					if err := s.sendToClient([]byte(reason), remote); err != nil {
						s.logger.Log(logger.LogEntry{Info: err.Error()})
						return
					}
					s.logger.Log(logger.LogEntry{Info: err.Error()})
					return
				}

				var response bytes.Buffer
				header, err := message.HandleHeader(buffer[:12])
				if err != nil {
					if err := s.sendToClient([]byte(err.Error()), remote); err != nil {
						s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						return
					}
					s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
					return
				}

				s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Request header: %v", header)})
				s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("questions: %d", header.Qdcount)})

				questions, n, err := message.HandleQuestions(buffer[:b], header.Qdcount, s.cache)
				if err != nil {
					if err := s.sendToClient([]byte(err.Error()), remote); err != nil {
						return
					}
					s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
					return
				}

				builder := message.NewResponseBuilder()

				switch n {
				case 0:
					GoogleAnswer, err := GoogleDNS.RequestToGoogleDNS(c, buffer)
					if err != nil {
						if err := s.sendToClient([]byte(err.Error()), remote); err != nil {
							s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						}
						s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						return
					}

					if _, err := response.Write(GoogleAnswer); err != nil {
						if err := s.sendToClient([]byte(err.Error()), remote); err != nil {
							s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						}
						s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						return
					}

					s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Google: %v", GoogleAnswer)})
				case 1:
					s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Cache questions: %v", questions[0])})

					header.Ancount = 1
					header.Qdcount = 1
					header.Arcount = 0
					header.Nscount = 0

					header.SetFlags(1, 0, 0, 0, 0, 1, 0, 0)
					resp := builder.BuildResponse(header, questions[0], s.cache)
					if resp.Err != nil {
						if err := s.sendToClient([]byte(resp.Err.Error()), remote); err != nil {
							s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						}
						s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: response builder error: %s", err)})
						return
					}

					if _, err := response.Write(resp.Data); err != nil {
						if err := s.sendToClient([]byte(err.Error()), remote); err != nil {
							s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						}
						s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
						return
					}

				default:
					for _, que := range questions {
						s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Cache question: %v", que)})
					}

					header.SetFlags(1, 0, 0, 0, 1, 1, 0, 0)
					header.Qdcount = uint16(n)

					wg := &sync.WaitGroup{}
					respChan := make(chan message.Response, n)
					for _, que := range questions {
						wg.Add(1)
						go func(que message.Question) {
							defer wg.Done()
							respChan <- builder.BuildResponse(header, que, s.cache)
						}(que)
					}

					go func() {
						close(respChan)
						wg.Wait()
					}()

					for resp := range respChan {
						if resp.Err != nil {
							if err := s.sendToClient([]byte(resp.Err.Error()), remote); err != nil {
								s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
							}
							s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Builder response: %v", err)})
							continue
						}

						if _, err := response.Write(resp.Data); err != nil {
							s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("response error: %s", err.Error())})
						}
					}
				}

				if err := s.sendToClient(response.Bytes(), remote); err != nil {
					s.logger.Log(logger.LogEntry{Info: fmt.Sprintf("Error: %v", err)})
					return
				}
			}

		}(context.Background(), buf, b, remote)

	}
}

func (s *Server) CloseUDP() error {
	timer := time.NewTimer(time.Second * 15)
	defer timer.Stop()

	select {
	case <-timer.C:
		if timer.Stop() {
			return fmt.Errorf("timeout")
		}
	default:
		s.limit.Close()
		s.cache.Close()
		if err := s.udpConn.Close(); err != nil {
			return err
		}
	}

	return nil
}
