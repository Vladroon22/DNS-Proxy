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
}

func NewLimiter(rps int) *Limiter {
	return &Limiter{
		incomingIps: make(map[string]*rpsCounter),
		bannedIPs:   make(map[string]time.Time),
		mtx:         sync.RWMutex{},
		limit:       rps,
	}
}

func (l *Limiter) StartLimiter() {
	l.cleanIncomingIPs()
	l.cleanBannedIPs()
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

func (l *Limiter) cleanBannedIPs() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ticker.C:
			l.mtx.RLock()
			for ip, tm := range l.bannedIPs {
				if time.Since(tm) <= 0.0 {
					delete(l.bannedIPs, ip)
				}
			}
			l.mtx.RUnlock()
		default:
			continue
		}
	}
}

func (l *Limiter) cleanIncomingIPs() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ticker.C:
			l.mtx.RLock()
			defer l.mtx.RUnlock()
			for ip, rps := range l.incomingIps {
				if rps.lastRequest.Before(time.Now().Add(-time.Second)) {
					delete(l.incomingIps, ip)
				}
			}
		default:
			continue
		}
	}
}
