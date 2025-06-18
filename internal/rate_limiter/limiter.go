package rate_limiter

import (
	"strconv"
	"sync"
	"time"
)

const (
	ErrBanIP     = "access denied"
	ErrExceedRPS = "exceed requests per second"
)

type Limiter struct {
	incomingIps map[string]*rpsCounter
	bannedIPs   map[string]time.Duration
	mtx         sync.Mutex
	limit       int
}

func NewLimiter(rps int) *Limiter {
	return &Limiter{
		incomingIps: make(map[string]*rpsCounter),
		bannedIPs:   make(map[string]time.Duration),
		mtx:         sync.Mutex{},
		limit:       rps,
	}
}

func (l *Limiter) StartLimiter() {
	if len(l.bannedIPs) > 0 {
		l.cleanBannedIPs()
	}
}

func (l *Limiter) setIP(ip string) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.bannedIPs == nil {
		l.bannedIPs = make(map[string]time.Duration)
	}
	l.bannedIPs[ip] = time.Minute * 5
}

func (l *Limiter) ProcessIP(ip string) (bool, string) {
	l.mtx.Lock()
	tm, ban := l.bannedIPs[ip]
	l.mtx.Unlock()
	if ban && int(tm.Seconds()) > 0 {
		return true, ErrBanIP + " for " + tm.String()
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()
	defer delete(l.incomingIps, ip)

	RPS := 0
	rpscounter, ok := l.incomingIps[ip]
	if !ok {
		l.incomingIps[ip] = NewRPSCounter()
		l.incomingIps[ip].increment()
		RPS = int(l.incomingIps[ip].getRPS())
	} else {
		rpscounter.increment()
		RPS = int(rpscounter.getRPS())
	}

	if RPS > l.limit {
		l.setIP(ip)
		return true, ErrExceedRPS + " " + strconv.Itoa(RPS)
	}

	return false, ""
}

func (l *Limiter) cleanBannedIPs() {
	for l.bannedIPs != nil {
		l.mtx.Lock()

		for ip, tm := range l.bannedIPs {
			if int(tm.Seconds()) <= 0 {
				delete(l.bannedIPs, ip)
			}
		}

		l.mtx.Unlock()
		time.Sleep(time.Second * 5)
	}
}
