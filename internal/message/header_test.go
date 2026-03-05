package message

import (
	"encoding/binary"
	"testing"
)

func TestFlagsBinaryRepresentation(t *testing.T) {
	h := Header{}
	var flags uint16

	// Тестируем конкретный случай: QR=1, RD=1, RA=1
	flags = h.SetFlags(1, 0, 0, 0, 1, 1, 0, 0)

	// Должно быть: 10000000 00000000 в бинарном
	// QR=1 (бит 15), RD=1 (бит 8), RA=1 (бит 7)
	expectedBinary := uint16(0x8000 | 0x0100 | 0x0080) // 0x8180
	if flags != expectedBinary {
		t.Errorf("Binary representation incorrect: expected %016b (%04x), got %016b (%04x)", expectedBinary, expectedBinary, flags, flags)
	}

	// Проверяем парсинг
	QR, OPcode, AA, TC, RD, RA, Z, Rcode := h.parseFlags()

	if QR != 1 {
		t.Errorf("QR expected 1, got %d", QR)
	}
	if RD != 1 {
		t.Errorf("RD expected 1, got %d", RD)
	}
	if RA != 1 {
		t.Errorf("RA expected 1, got %d", RA)
	}
	if OPcode != 0 || AA != 0 || TC != 0 || Z != 0 || Rcode != 0 {
		t.Errorf("Unexpected flags: OPcode=%d, AA=%d, TC=%d, Z=%d, Rcode=%d",
			OPcode, AA, TC, Z, Rcode)
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	// Тест полного цикла: Header → байты → Header
	original := Header{
		ID:      0xABCD,
		Flags:   0,
		Qdcount: 1,
		Ancount: 2,
		Nscount: 0,
		Arcount: 1,
	}

	original.SetFlags(1, 0, 1, 0, 1, 1, 0, 0)

	encoded, _ := original.Decode()

	decoded, err := HandleHeader(encoded)
	if err != nil {
		t.Fatalf("HandleHeader failed: %v", err)
	}

	if original.ID != decoded.ID {
		t.Errorf("ID mismatch: original %04x, decoded %04x", original.ID, decoded.ID)
	}

	if original.Flags != decoded.Flags {
		t.Errorf("Flags mismatch: original %04x, decoded %04x", original.Flags, decoded.Flags)
	}

	if original.Qdcount != decoded.Qdcount {
		t.Errorf("Qdcount mismatch: original %d, decoded %d", original.Qdcount, decoded.Qdcount)
	}

	QR, _, AA, _, RD, RA, _, _ := decoded.parseFlags()

	if QR != 1 {
		t.Errorf("Decoded QR incorrect: expected 1, got %d", QR)
	}

	if AA != 1 {
		t.Errorf("Decoded AA incorrect: expected 1, got %d", AA)
	}

	if RD != 1 {
		t.Errorf("Decoded RD incorrect: expected 1, got %d", RD)
	}

	if RA != 1 {
		t.Errorf("Decoded RA incorrect: expected 1, got %d", RA)
	}
}

func TestParseFlagsFromRealDNS(t *testing.T) {
	dnsPacket := []byte{
		0x12, 0x34, // ID
		0x81, 0x80, // Flags = 0x8180 (QR=1, RD=1, RA=1)
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x02, // ANCOUNT = 2
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x01, // ARCOUNT = 1
	}

	h, err := HandleHeader(dnsPacket)
	if err != nil {
		t.Fatalf("HandleHeader failed: %v", err)
	}

	if h.ID != 0x1234 {
		t.Errorf("ID incorrect: expected 0x1234, got 0x%04x", h.ID)
	}

	QR, _, _, _, RD, RA, _, _ := h.parseFlags()

	if QR != 1 {
		t.Errorf("QR incorrect: expected 1, got %d", QR)
	}

	if RD != 1 {
		t.Errorf("RD incorrect: expected 1, got %d", RD)
	}

	if RA != 1 {
		t.Errorf("RA incorrect: expected 1, got %d", RA)
	}

	if h.Flags != 0x8180 {
		t.Errorf("Flags value incorrect: expected 0x8180, got 0x%04x", h.Flags)
	}
}

func TestErrorConditions(t *testing.T) {
	shortData := []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03} // 6 bytes
	if _, err := HandleHeader(shortData); err == nil {
		t.Error("Expected error for short header, got nil")
	} else if err.Error() != ErrShortMsg {
		t.Errorf("Expected '%s', got: %v", ErrShortMsg, err)
	}

	zeroQdcount := make([]byte, 12)
	binary.BigEndian.PutUint16(zeroQdcount[4:6], 0) // QDCOUNT = 0
	if _, err := HandleHeader(zeroQdcount); err == nil {
		t.Error("Expected error for QDCOUNT=0, got nil")
	} else if err.Error() != ErrQdcountZero {
		t.Errorf("Expected '%s', got: %v", ErrQdcountZero, err)
	}
}

func BenchmarkSetFlags(b *testing.B) {
	h := Header{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.SetFlags(1, 0, 0, 0, 1, 1, 0, 0)
	}
}

func BenchmarkParseFlags(b *testing.B) {
	h := Header{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.parseFlags()
	}
}
