package rate_limiter

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	banTime      = time.Minute * 5
	ErrBanIP     = "access denied"
	ErrExceedRPS = "exceed req/sec"
)

type Limiter struct {
	incomingIps map[string]*rpsCounter
	bannedIPs   map[string]time.Time
	mtx         sync.Mutex
	limit       int
}

func NewLimiter(rps int) *Limiter {
	return &Limiter{
		incomingIps: make(map[string]*rpsCounter),
		bannedIPs:   make(map[string]time.Time),
		mtx:         sync.Mutex{},
		limit:       rps,
	}
}

func (l *Limiter) StartLimiter() {
	l.cleanIncomingIPs()
	if len(l.bannedIPs) > 0 {
		l.cleanBannedIPs()
	}
}

func (l *Limiter) setIP(ip string) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.bannedIPs == nil {
		l.bannedIPs = make(map[string]time.Time)
	}
	l.bannedIPs[ip] = time.Now().Add(banTime)
}

func (l *Limiter) ProcessIP(ip string) (bool, string) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if tm, banned := l.bannedIPs[ip]; banned && time.Since(tm) > 0 {
		return true, fmt.Sprintf("%s for %s", ErrBanIP, tm.String())
	}

	RPS := 0
	if _, ok := l.incomingIps[ip]; !ok {
		l.incomingIps[ip] = NewRPSCounter()
	}
	l.incomingIps[ip].increment()
	RPS = int(l.incomingIps[ip].getRPS())

	if RPS > l.limit {
		l.setIP(ip)
		return true, fmt.Sprintf("%s: %s", ErrExceedRPS, strconv.Itoa(RPS))
	}

	return false, ""
}

func (l *Limiter) cleanBannedIPs() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		l.mtx.Lock()

		for ip, tm := range l.bannedIPs {
			if time.Since(tm) <= 0.0 {
				delete(l.bannedIPs, ip)
			}
		}

		l.mtx.Unlock()
	}
}

func (l *Limiter) cleanIncomingIPs() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		l.mtx.Lock()

		for ip, rps := range l.incomingIps {
			if rps.lastRequest.Before(time.Now().Add(-time.Second)) {
				delete(l.incomingIps, ip)
			}
		}

		l.mtx.Unlock()
	}
}
