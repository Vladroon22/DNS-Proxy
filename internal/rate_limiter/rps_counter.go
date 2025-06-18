package rate_limiter

import (
	"sync"
	"sync/atomic"
	"time"
)

type rpsCounter struct {
	mtx   sync.Mutex
	count uint32
	tick  *time.Ticker
}

func NewRPSCounter() *rpsCounter {
	rps := &rpsCounter{
		mtx:   sync.Mutex{},
		count: 0,
		tick:  time.NewTicker(time.Second),
	}
	go rps.reset()
	return rps
}

func (rps *rpsCounter) reset() {
	for range rps.tick.C {
		rps.mtx.Lock()
		rps.count = 0
		rps.mtx.Unlock()
	}
}

func (rps *rpsCounter) increment() {
	atomic.AddUint32(&rps.count, 1)
}

func (rps *rpsCounter) getRPS() uint32 {
	return atomic.LoadUint32(&rps.count)
}
