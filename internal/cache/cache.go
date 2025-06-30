package cache

import (
	"net"
	"sync"
	"time"
)

type Item struct {
	Class  uint16
	Type   uint16
	Length uint16
	IP     net.IP
	Name   string
	Exp    time.Time
}

type Cache struct {
	cache map[string]*Item
	mtx   sync.RWMutex
}

func InitCache() *Cache {
	chc := &Cache{
		cache: make(map[string]*Item),
		mtx:   sync.RWMutex{},
	}
	go chc.cleanRecords()

	return chc
}

func (c *Cache) Set(ip []byte, name string, class, tp, len uint16, ttl uint32) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.cache == nil {
		c.cache = make(map[string]*Item)
	}

	item := &Item{
		IP:     ip,
		Name:   name,
		Class:  class,
		Type:   tp,
		Length: len,
		Exp:    time.Now().Add(time.Duration(ttl) * time.Second),
	}

	c.cache[name] = item
}

func (c *Cache) Get(tp uint16, dmn string) (Item, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if item, ok := c.cache[dmn]; ok {
		if item.Exp.After(time.Now()) {
			return *item, true
		}
	}

	return Item{}, false
}

func (c *Cache) cleanRecords() {
	for c.cache != nil {
		c.mtx.Lock()

		for _, item := range c.cache {
			if item.Exp.Second() <= 0 {
				delete(c.cache, item.Name)
			}
		}

		c.mtx.Unlock()
		time.Sleep(time.Second * 10)
	}
}
