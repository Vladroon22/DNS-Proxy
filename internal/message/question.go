package message

import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Vladroon22/DNS-Server/internal/cache"
)

type QType uint16

const (
	_     QType      = iota
	A                // a host address
	NS               // an authoritative name server
	MD               // a mail destination
	MF               // a mail forwarder
	CNAME            // the canonical name for an alias
	SOA              // marks the start of a zone of authority
	MB               // a mailbox domain name (EXPERIMENTAL)
	MG               // a mail group member (EXPERIMENTAL)
	MR               // a mail rename domain name (EXPERIMENTAL)
	NULL             // a null RR (EXPERIMENTAL)
	WKS              // a well known service description
	PTR              // a domain name pointer
	HINFO            // host information
	MINFO            // mailbox or mail list information
	MX               // mail exchange
	TXT              // text strings
	AAAA  = TXT + 12 // ipv6 address
)

type QClass uint16

const (
	_  QClass = iota
	IN        // the Internet
	CS        // the CSNET class (Obsolete - used only for examples in some obsolete RFCs)
	CH        // the CHAOS class
	HS        // Hesiod [Dyer 87]
)

type Question struct {
	Name  string
	Type  QType
	Class QClass
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

func HandleQuestions(data []byte, qdcount uint16, che *cache.Cache) ([]Question, int, error) {
	questions := make([]Question, 0, qdcount)
	offset := 12

	for range qdcount {
		var name string
		var err error
		name, offset, err = readName(data, offset, data)
		if err != nil {
			return nil, 0, err
		}

		qtype := binary.BigEndian.Uint16(data[offset : offset+2])
		qclass := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		questions = append(questions, Question{Name: name, Type: QType(qtype), Class: QClass(qclass)})
		offset += 4
	}

	wg := sync.WaitGroup{}
	n := len(questions)
	log.Println("len ques:", n)
	questionsCh := make(chan Question, n)

	for _, que := range questions {
		log.Println(que)
		wg.Add(1)
		go func(que *Question) {
			if _, ok := che.Get(uint16(que.Type), que.Name); ok {
				questionsCh <- *que
			}
			wg.Done()
		}(&que)
	}

	go func() {
		wg.Wait()
		close(questionsCh)
	}()

	questions = questions[:0]
	for que := range questionsCh {
		questions = append(questions, que)
	}

	return questions, len(questions), nil
}
