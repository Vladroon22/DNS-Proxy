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
	cache  map[string]*Item
	mtx    sync.RWMutex
	exitCh chan struct{}
}

func InitCache() *Cache {
	chc := &Cache{
		cache:  make(map[string]*Item),
		mtx:    sync.RWMutex{},
		exitCh: make(chan struct{}),
	}
	go chc.cleanRecords()

	return chc
}

func (c *Cache) Set(ip []byte, name string, class, tp, len uint16, ttl uint32) {
	if c.cache == nil {
		c.cache = make(map[string]*Item)
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

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
		if item.Type == tp && item.Exp.After(time.Now()) {
			return *item, true
		}
	}

	return Item{}, false
}

func (c *Cache) cleanRecords() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanExpired()
		case <-c.exitCh:
			return
		}
	}
}

func (c *Cache) cleanExpired() {
	now := time.Now()

	c.mtx.Lock()
	defer c.mtx.Unlock()

	for name, item := range c.cache {
		if item.Exp.Before(now) {
			delete(c.cache, name)
		}
	}
}

func (c *Cache) Close() {
	c.exitCh <- struct{}{}
}
