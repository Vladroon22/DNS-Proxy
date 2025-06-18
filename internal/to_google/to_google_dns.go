package to_google

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/Vladroon22/DNS-Server/internal/cache"
	"github.com/Vladroon22/DNS-Server/internal/message"
)

const (
	v1 = "8.8.8.8:53"
	v2 = "8.8.4.4:53"
	v3 = "1.1.1.1:53"
	v4 = "8.8.8.8:853"
)

func RequestToGoogleDNS(request []byte, qdcount int, che *cache.Cache, eDNS bool) ([]byte, error) {
	var size int
	var netType string
	if eDNS {
		size = 4096
		netType = "tcp"
	} else {
		size = 512
		netType = "udp"
	}

	var conn net.Conn
	for _, dns := range []string{v1, v2, v3, v4} {
		var err error
		conn, err = net.DialTimeout(netType, dns, 5*time.Second)
		if err == nil {
			break
		}
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	_, errW := conn.Write(request)
	if errW != nil {
		log.Println("Error of sending request to google:", errW)
		return nil, fmt.Errorf("error of sending request to google: %s", errW)
	}

	data := make([]byte, size)
	n, err := conn.Read(data)
	if err != nil {
		log.Println("Error reading answer from google:", err)
		return nil, fmt.Errorf("error reading answer from google: %s", err)
	}

	resp := data[12:n]
	if err := parseGoogleResponse(resp, qdcount, che); err != nil {
		log.Println("Error parse answer from google:", err)
		return nil, fmt.Errorf("error reading answer from google: %s", err)
	}

	return data[:n], nil
}

func parseGoogleResponse(data []byte, qdcount int, che *cache.Cache) error {
	start := 12

	for range qdcount {
		var name string
		name, offset := skipQuestion(data, start) // здесь проблема
		if offset+10 > len(data) {
			return fmt.Errorf("incorrect record format")
		}

		Type := binary.BigEndian.Uint16(data[offset : offset+2])
		Class := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		ttl := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		length := binary.BigEndian.Uint16(data[offset+8 : offset+10])
		offset += 10

		log.Println(name)
		log.Println(Class)
		log.Println(Type)
		log.Println(length)
		log.Println(ttl)

		if offset+int(length) > len(data) {
			return fmt.Errorf("malformed DNS packet: answer data too short")
		}

		var ip []byte
		switch Type {
		case uint16(message.A):
			ip = data[offset : offset+3]
		case uint16(message.AAAA):
			ip = data[offset : offset+16]
		default:
			return fmt.Errorf("unsupported type of record")
		}
		che.Set(ip, name, Class, Type, length, ttl)
	}

	return nil
}

func skipQuestion(data []byte, offset int) (string, int) {
	domain := []string{}
	visit := make(map[int]bool)

	for {
		length := int(data[offset])
		if length == 0 {
			break
		}

		if length&0xC0 == 0xC0 {
			if visit[offset] {
				break
			}

			visit[offset] = true
			if len(domain) > 0 {
				break
			}
			break
		}

		offset++
		domain = append(domain, string(data[offset:offset+length]))
		offset += length
	}
	name := strings.Join(domain, ".")
	offset++

	if offset+4 > len(data) {
		return name, offset
	}
	offset += 4

	return name, offset
}

func SkipQuestion(data []byte, offset int) (string, int) {
	domain := []string{}
	visit := make(map[int]bool)

	for {
		length := int(data[offset])
		if length == 0 {
			break
		}

		if length&0xC0 == 0xC0 {
			if visit[offset] {
				break
			}

			visit[offset] = true
			if len(domain) > 0 {
				break
			}
			break
		}

		offset++
		domain = append(domain, string(data[offset:offset+length]))
		offset += length
	}
	name := strings.Join(domain, ".")
	offset++

	offset += 4

	return name, offset
}
