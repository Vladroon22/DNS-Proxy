package message

import (
	"encoding/binary"
	"strings"
	"sync"

	"github.com/Vladroon22/DNS-Server/internal/cache"
)

type Type uint16

const (
	_     Type       = iota
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

type Class uint16

const (
	_  Class = iota
	IN       // the Internet
	CS       // the CSNET class (Obsolete - used only for examples in some obsolete RFCs)
	CH       // the CHAOS class
	HS       // Hesiod [Dyer 87]
)

type Question struct {
	Name  string
	Type  Type
	Class Class
}

func handleQuestion(data []byte, offset int) (Question, int) {
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

	qtype := binary.BigEndian.Uint16(data[offset : offset+2])
	qclass := binary.BigEndian.Uint16(data[offset+2 : offset+4])

	que := Question{
		Name:  name,
		Type:  Type(qtype),
		Class: Class(qclass),
	}

	return que, offset
}

func HandleQuestions(data []byte, qdcount uint16, che *cache.Cache) ([]Question, int) {
	questions := make([]Question, 0, qdcount)
	offset := 12

	for range qdcount {
		que, newOffset := handleQuestion(data, offset)
		questions = append(questions, que)
		offset = newOffset + 4
	}

	wg := sync.WaitGroup{}
	n := len(questions)
	questionsCh := make(chan Question, n)

	for _, que := range questions {
		wg.Add(1)
		go func(que Question) {
			defer wg.Done()
			if _, ok := che.Get(uint16(que.Type), que.Name); !ok {
				return
			}
			questionsCh <- que
		}(que)
	}

	go func() {
		wg.Wait()
		close(questionsCh)
	}()

	if len(questionsCh) == 0 {
		return nil, 0
	}

	questions = questions[:0]
	for que := range questionsCh {
		questions = append(questions, que)
	}

	return questions, len(questions)
}
