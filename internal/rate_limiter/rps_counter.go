package rate_limiter

import (
	"sync"
	"time"
)

type rpsCounter struct {
	mtx         sync.Mutex
	count       uint32
	lastRequest time.Time
}

func NewRPSCounter() *rpsCounter {
	rps := &rpsCounter{
		mtx:   sync.Mutex{},
		count: 0,
	}
	go rps.reset()
	return rps
}

func (rps *rpsCounter) reset() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for range tick.C {
		rps.mtx.Lock()
		rps.count = 0
		rps.mtx.Unlock()
	}
	rps.lastRequest = time.Now()
}

func (rps *rpsCounter) increment() {
	rps.mtx.Lock()
	defer rps.mtx.Unlock()
	rps.count++
}

func (rps *rpsCounter) getRPS() uint32 {
	rps.mtx.Lock()
	defer rps.mtx.Unlock()
	return rps.count
}
