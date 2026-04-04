package rate_limiter

import (
	"fmt"
	"sync"
	"time"
)

const (
	BanTime      = time.Minute * 5
	ErrBanIP     = "access denied"
	ErrExceedRPS = "exceed req/sec"
)

type Limiter struct {
	incomingIps map[string]*rpsCounter
	bannedIPs   map[string]time.Time
	mtx         sync.RWMutex
	limit       int
	exitCh      chan struct{}
}

func NewLimiter(rps int) *Limiter {
	return &Limiter{
		incomingIps: make(map[string]*rpsCounter),
		bannedIPs:   make(map[string]time.Time),
		mtx:         sync.RWMutex{},
		limit:       rps,
		exitCh:      make(chan struct{}),
	}
}

func (l *Limiter) StartLimiter() {
	l.cleanIPs()
}

func (l *Limiter) ProcessIP(ip string) (bool, string) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if tm, banned := l.bannedIPs[ip]; banned && time.Since(tm) > 0 {
		return true, fmt.Sprintf("%s for %s", ErrBanIP, tm.String())
	}

	var counter *rpsCounter
	counter, ok := l.incomingIps[ip]
	if !ok {
		counter = NewRPSCounter()
		l.incomingIps[ip] = counter
	}
	counter.increment()

	rps := int(counter.getRPS())
	if rps > l.limit {
		if l.bannedIPs == nil {
			l.bannedIPs = make(map[string]time.Time)
		}
		l.bannedIPs[ip] = time.Now().Add(BanTime)
		return true, fmt.Sprintf("%s: %d", ErrExceedRPS, rps)
	}

	return false, ""
}

func (l *Limiter) cleanIPs() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanIncomingIPs()
			l.cleanBannedIPs()
		case <-l.exitCh:
			return
		}
	}
}

func (l *Limiter) cleanBannedIPs() {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	for ip, tm := range l.bannedIPs {
		if time.Since(tm) <= 0.0 {
			delete(l.bannedIPs, ip)
		}
	}
}

func (l *Limiter) cleanIncomingIPs() {
	now := time.Now()

	l.mtx.Lock()
	defer l.mtx.Unlock()

	for ip, rps := range l.incomingIps {
		if rps.lastRequest.Before(now.Add(-time.Second)) {
			delete(l.incomingIps, ip)
		}
	}
}

func (l *Limiter) Close() {
	l.exitCh <- struct{}{}
}
