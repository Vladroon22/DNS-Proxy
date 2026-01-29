package compress

import (
	"bytes"
	"encoding/binary"
	"strings"
	"sync"
)

type domainInfo struct {
	offset  int
	pointer int
}

type Compress struct {
	mu    sync.RWMutex
	names map[string]*domainInfo
}

func NewCompress() *Compress {
	return &Compress{
		names: make(map[string]*domainInfo),
	}
}

func (c *Compress) AddName(name string, offset int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if info, exists := c.names[name]; exists {
		info.pointer++
		return
	}

	c.names[name] = &domainInfo{
		offset:  offset,
		pointer: 0,
	}
}

func (c *Compress) EncodeName(name string, currOffset int) []byte {
	if name == "" || name == "." {
		return []byte{0}
	}

	name = strings.TrimSuffix(name, ".")

	c.mu.RLock()
	info, exists := c.names[name]
	c.mu.RUnlock()

	if exists && info.offset < currOffset {
		c.mu.Lock()
		info.pointer++
		c.mu.Unlock()

		pointer := make([]byte, 2)
		binary.BigEndian.PutUint16(pointer, uint16(0xC000|info.offset))
		return pointer
	}

	var buf bytes.Buffer
	labels := strings.Split(name, ".")

	for _, label := range labels {
		if len(label) > 63 {
			label = label[:63]
		}
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0)

	c.AddName(name, currOffset)

	return buf.Bytes()
}

func (c *Compress) GetPointerCount(name string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if info, exists := c.names[name]; exists {
		return info.pointer
	}
	return 0
}

func (c *Compress) GetOffset(name string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if info, exists := c.names[name]; exists {
		return info.offset, true
	}
	return 0, false
}

func (c *Compress) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.names = make(map[string]*domainInfo)
}

func (c *Compress) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.names)
}
