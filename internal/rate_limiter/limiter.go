package rate_limiter

import (
	"context"
	"fmt"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l.cleanIncomingIPs(ctx)
	if len(l.bannedIPs) > 0 {
		l.cleanBannedIPs(ctx)
	}
}

func (l *Limiter) setIP(ip string) {
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

	var counter *rpsCounter
	counter, ok := l.incomingIps[ip]
	if !ok {
		counter = NewRPSCounter()
		l.incomingIps[ip] = counter
	}
	counter.increment()

	rps := int(counter.getRPS())
	if rps > l.limit {
		l.setIP(ip)
		return true, fmt.Sprintf("%s: %d", ErrExceedRPS, rps)
	}

	return false, ""
}

func (l *Limiter) cleanBannedIPs(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mtx.Lock()
			for ip, tm := range l.bannedIPs {
				if time.Since(tm) <= 0.0 {
					delete(l.bannedIPs, ip)
				}
			}
			l.mtx.Unlock()
		case <-ctx.Done():
			return
		}
	}

}

func (l *Limiter) cleanIncomingIPs(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		l.mtx.Lock()
		defer l.mtx.Unlock()
		for ip, rps := range l.incomingIps {
			if rps.lastRequest.Before(time.Now().Add(-time.Second)) {
				delete(l.incomingIps, ip)
			}
		}
	case <-ctx.Done():
		return
	}
}
