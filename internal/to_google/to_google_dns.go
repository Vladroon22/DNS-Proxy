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
	DNSv1 = "8.8.8.8:53"
	DNSv2 = "8.8.4.4:53"
	DNSv3 = "1.1.1.1:53"
	DNSv4 = "8.8.8.8:853"
)

type DNSReceiver struct {
	msg_size int
	network  string
	eDNS     bool
	che      *cache.Cache
}

func NewDNSReceiver(che *cache.Cache, size int, eDNS bool) *DNSReceiver {
	return &DNSReceiver{
		eDNS:     eDNS,
		msg_size: size,
		che:      che,
	}
}

func (rcv *DNSReceiver) RequestToGoogleDNS(request []byte) ([]byte, error) {
	if rcv.eDNS {
		rcv.msg_size = 4096
		rcv.network = "tcp"
	} else {
		rcv.msg_size = 512
		rcv.network = "udp"
	}

	var conn net.Conn
	for _, dns := range []string{DNSv1, DNSv2, DNSv3, DNSv4} {
		var err error
		conn, err = net.DialTimeout(rcv.network, dns, 5*time.Second)
		if err == nil {
			break
		}
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(request); err != nil {
		log.Println("Error of sending request to google:", err)
		return nil, fmt.Errorf("error of sending request to google: %s", err)
	}

	data := make([]byte, rcv.msg_size)
	n, err := conn.Read(data)
	if err != nil {
		log.Println("Error reading answer from google:", err)
		return nil, fmt.Errorf("error reading answer from google: %s", err)
	}

	if err := rcv.parseGoogleResponse(data[:n]); err != nil {
		log.Println("Error parse answer from google:", err)
		return nil, fmt.Errorf("error reading answer from google: %s", err)
	}

	return data[:n], nil
}

func (rcv *DNSReceiver) parseGoogleResponse(data []byte) error {
	header, err := message.HandleHeader(data[:12])
	if err != nil {
		return err
	}

	if rcv.che == nil {
		return fmt.Errorf("nil che")
	}

	offset := 12
	for range header.Qdcount {
		var err error
		_, offset, err = readName(data, offset, data)
		if err != nil {
			return err
		}
		offset += 4 // skip QClass and QType
	}

	for range header.Ancount {
		name, newOffset, err := readName(data, offset, data)
		if err != nil {
			return err
		}
		offset = newOffset

		if offset+10 > len(data) {
			return fmt.Errorf("wrong answer's format")
		}

		Type := binary.BigEndian.Uint16(data[offset : offset+2])
		Class := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		ttl := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		length := binary.BigEndian.Uint16(data[offset+8 : offset+10])
		offset += 10

		log.Println("name:", name)
		log.Println("class:", Class)
		log.Println("type:", Type)
		log.Println("len:", length)
		log.Println("ttl:", ttl)

		var ip []byte
		switch Type {
		case uint16(message.A), uint16(message.AAAA):
			ip = data[offset : offset+int(length)]
		default:
			return fmt.Errorf("unsupported type of record")
		}
		log.Println("ip:", ip)
		rcv.che.Set(ip, name, Class, Type, length, ttl)
		offset += int(length)
	}

	return nil
}

func readName(data []byte, offset int, origData []byte) (string, int, error) {
	names := []string{}
	visit := make(map[int]bool)

	for offset < len(data) {
		length := int(data[offset])
		if length == 0 {
			return strings.Join(names, "."), offset + 1, nil
		}

		if length&0xC0 == 0xC0 {
			if offset+1 >= len(data) {
				return "", offset, fmt.Errorf("invalid compression pointer")
			}
			ptr := int(binary.BigEndian.Uint16(data[offset:offset+2]) & 0x3FFF)
			if visit[ptr] {
				return "", offset, fmt.Errorf("compression is cycled")
			}
			visit[ptr] = true
			name, _, err := readName(data[ptr:], 0, origData)
			if err != nil {
				return "", offset, err
			}
			names = append(names, name)
			return strings.Join(names, "."), offset + 2, nil
		}
		if offset+1+length > len(data) {
			return "", offset, fmt.Errorf("invalid label length")
		}
		names = append(names, string(data[offset+1:offset+1+length]))
		offset += 1 + length
	}
	return "", offset, fmt.Errorf("unexpected EOF")
}
