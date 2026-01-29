package message

import (
	"bytes"
	"encoding/binary"
	"log"
	"sync"

	"github.com/Vladroon22/DNS-Server/internal/cache"
	"github.com/Vladroon22/DNS-Server/internal/compress"
)

type ResponseBuilder struct {
	buffer *bytes.Buffer
	cmp    *compress.Compress
	mtx    sync.RWMutex
}

func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		cmp:    compress.NewCompress(),
		buffer: bytes.NewBuffer(nil),
	}
}

type Response struct {
	Data []byte
	Err  error
}

func (rb *ResponseBuilder) BuildResponse(header *Header, que Question, che *cache.Cache) Response {
	header.Ancount++

	decodedHeader, err := header.Decode()
	if err != nil {
		return Response{}
	}

	if _, err := rb.buffer.Write(decodedHeader); err != nil {
		return Response{}
	}

	rb.cmp.AddName("", 12)
	log.Println("DNS response header:", header)

	rb.buffer.Write(rb.cmp.EncodeName(que.Name, rb.buffer.Len()))
	if err := binary.Write(rb.buffer, binary.BigEndian, que.Type); err != nil {
		return Response{}
	}

	if err := binary.Write(rb.buffer, binary.BigEndian, que.Class); err != nil {
		return Response{}
	}

	log.Println("DNS response question:", que)

	record, _ := che.Get(uint16(que.Type), que.Name)
	rb.buffer.Write([]byte{0xC0, 0x0C})

	if err := binary.Write(rb.buffer, binary.BigEndian, que.Type); err != nil {
		return Response{}
	}

	if err := binary.Write(rb.buffer, binary.BigEndian, que.Class); err != nil {
		return Response{}
	}

	ttl := make([]byte, 4)
	binary.BigEndian.PutUint32(ttl, uint32(record.Exp.Second()))
	if err := binary.Write(rb.buffer, binary.BigEndian, ttl); err != nil {
		return Response{}
	}

	if err := binary.Write(rb.buffer, binary.BigEndian, record.Length); err != nil {
		return Response{}
	}
	if _, err := rb.buffer.Write(record.IP); err != nil {
		return Response{}
	}

	total := rb.buffer.Bytes()
	rb.buffer.Reset()

	return Response{Data: total, Err: nil}
}
