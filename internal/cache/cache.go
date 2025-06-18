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
	Exp    time.Duration
}

type Cache struct {
	cache map[uint16]map[string]*Item
	mtx   sync.RWMutex
}

func InitCache() *Cache {
	chc := &Cache{
		cache: make(map[uint16]map[string]*Item),
		mtx:   sync.RWMutex{},
	}
	go chc.cleanRecords()

	return chc
}

func (c *Cache) Set(ip []byte, name string, class, tp, len uint16, ttl uint32) {
	if c.cache == nil {
		c.cache = make(map[uint16]map[string]*Item)
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	itm := &Item{
		IP:     ip,
		Name:   name,
		Class:  class,
		Type:   tp,
		Length: len,
		Exp:    time.Duration(ttl),
	}

	c.cache[tp][name] = itm
}

func (c *Cache) Get(tp uint16, dmn string) (Item, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if cache, ok := c.cache[tp]; ok {
		if itm, ok := cache[dmn]; ok {
			if int(itm.Exp.Seconds()) > 0 {
				return *itm, true
			}
		}
	}

	return Item{}, false
}

func (c *Cache) cleanRecords() {
	for c.cache != nil {
		c.mtx.Lock()

		for _, chc := range c.cache {
			for dmn, itm := range chc {
				if int(itm.Exp.Seconds()) <= 0 {
					delete(chc, dmn)
				}
			}
		}

		c.mtx.Unlock()
		time.Sleep(time.Second * 10)
	}
}
