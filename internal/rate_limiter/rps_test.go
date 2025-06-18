package rate_limiter

import (
	"net"
	"os"
	"sync"
	"testing"
)

func TestRPSConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	requests := 1000

	dnsQuery := []byte{
		0x12, 0x34, // ID
		0x01, 0x00, // Flags
		0x00, 0x01, // Questions
		0x00, 0x00, // Answers
		0x00, 0x00, // Authorities
		0x00, 0x00, // Additional

		0x07, 'y', 'o', 'u', 't', 'u', 'b', 'e',
		0x03, 'c', 'o', 'm',
		0x00,

		0x00, 0x01, 0x00, 0x01,
	}

	wg.Add(requests)
	for range requests {
		go func() {
			defer wg.Done()

			conn, err := net.Dial("udp", ":"+os.Getenv("udp_port"))
			if err != nil {
				t.Errorf("%v\n", err)
			}
			defer conn.Close()

			if _, err := conn.Write([]byte(dnsQuery)); err != nil {
				t.Errorf("%v\n", err)
			}
		}()
	}

	wg.Wait()
}
