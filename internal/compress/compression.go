package compress

import (
	"bytes"
	"encoding/binary"
	"strings"
	"sync"
)

type Info struct {
	pointer int32
	offset  int
}

type Compress struct {
	names map[string]Info
	mu    sync.Mutex
}

func NewCompress() *Compress {
	return &Compress{
		names: make(map[string]Info),
		mu:    sync.Mutex{},
	}
}

func (cmp *Compress) AddName(name string, offset int) {
	cmp.mu.Lock()
	defer cmp.mu.Unlock()

	if _, ok := cmp.names[name]; !ok {
		cmp.names[name] = Info{offset: offset, pointer: 0}
	}
}

func (cmp *Compress) EncodeName(name string, currOffset int) []byte {
	cmp.mu.Lock()
	defer cmp.mu.Unlock()

	if info, ok := cmp.names[name]; ok && info.offset < currOffset {
		info.pointer++

		cmp.names[name] = info

		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(0xC000|info.offset))
		return buf
	}

	buf := bytes.NewBuffer(nil)
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) > 0 {
			buf.WriteByte(byte(len(label)))
			buf.WriteString(label)
		}
	}
	buf.WriteByte(0)
	cmp.AddName(name, currOffset)
	return buf.Bytes()
}
