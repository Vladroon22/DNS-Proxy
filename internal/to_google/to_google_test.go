package to_google

import (
	"net"
	"testing"
	"time"
)

func TestRequestToGoogle(t *testing.T) {
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

	var conn net.Conn
	for _, dns := range []string{v1, v2, v3, v4} {
		var err error
		conn, err = net.DialTimeout("udp", dns, time.Second*3)
		if err == nil {
			break
		} else {
			t.Errorf("%v\n", err)
		}
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(dnsQuery)); err != nil {
		t.Errorf("%v\n", err)
	}

	resp := make([]byte, 512)
	if _, err := conn.Read(resp); err != nil {
		t.Errorf("%v\n", err)
	}

}
