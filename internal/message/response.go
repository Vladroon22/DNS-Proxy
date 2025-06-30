package message

import (
	"bytes"
	"encoding/binary"
	"log"
	"sync"

	"github.com/Vladroon22/DNS-Server/internal/cache"
	"github.com/Vladroon22/DNS-Server/internal/compress"
)

/*
var cach = map[string]Answer{
	"youtube.com.": {
		Name:   "youtube.com.",
		Type:   A,
		Class:  IN,
		TTL:    300,
		Length: net.IPv4len,
		IP:     []byte{123, 123, 123, 123},
	},
}
*/

func BuildResponse(header *Header, que Question, che *cache.Cache) []byte {
	buffer := bytes.NewBuffer(nil)
	cmp := compress.NewCompress()
	mtx := sync.Mutex{}

	mtx.Lock()
	defer mtx.Unlock()

	header.Ancount++

	decodedHeader := header.Decode()
	buffer.Write(decodedHeader)
	cmp.AddName("", 12)
	log.Println("DNS response header:", header)

	buffer.Write(cmp.EncodeName(que.Name, buffer.Len()))
	binary.Write(buffer, binary.BigEndian, que.Type)
	binary.Write(buffer, binary.BigEndian, que.Class)
	log.Println("DNS response question:", que)

	record, _ := che.Get(uint16(que.Type), que.Name)
	buffer.Write([]byte{0xC0, 0x0C})

	binary.Write(buffer, binary.BigEndian, que.Type)
	binary.Write(buffer, binary.BigEndian, que.Class)

	ttl := make([]byte, 4)
	binary.BigEndian.PutUint32(ttl, uint32(record.Exp.Second()))
	binary.Write(buffer, binary.BigEndian, ttl)

	binary.Write(buffer, binary.BigEndian, record.Length)
	buffer.Write(record.IP)

	return buffer.Bytes()
}
