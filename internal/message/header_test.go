package message

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestFlagsBinaryRepresentation(t *testing.T) {
	h := Header{}
	var flags uint16

	// Тестируем конкретный случай: QR=1, RD=1, RA=1
	h.SetFlags(&flags, 1, 0, 0, 0, 1, 1, 0, 0)

	// Должно быть: 10000000 00000000 в бинарном
	// QR=1 (бит 15), RD=1 (бит 8), RA=1 (бит 7)
	expectedBinary := uint16(0x8000 | 0x0100 | 0x0080) // 0x8180
	if flags != expectedBinary {
		t.Errorf("Binary representation incorrect: expected %016b (%04x), got %016b (%04x)",
			expectedBinary, expectedBinary, flags, flags)
	}

	// Проверяем парсинг
	QR, OPcode, AA, TC, RD, RA, Z, Rcode := h.parseFlags(&flags)

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

	fmt.Printf("Test passed: Flags: %016b (hex: %04x)\n", flags, flags)
	fmt.Printf("QR=%d, RD=%d, RA=%d\n", QR, RD, RA)
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

	// Устанавливаем флаги
	original.SetFlags(&original.Flags, 1, 0, 1, 0, 1, 1, 0, 0)

	// Кодируем в байты
	encoded := original.Decode()

	// Декодируем обратно
	decoded, err := HandleHeader(encoded)
	if err != nil {
		t.Fatalf("HandleHeader failed: %v", err)
	}

	// Проверяем совпадение
	if original.ID != decoded.ID {
		t.Errorf("ID mismatch: original %04x, decoded %04x", original.ID, decoded.ID)
	}
	if original.Flags != decoded.Flags {
		t.Errorf("Flags mismatch: original %04x, decoded %04x", original.Flags, decoded.Flags)
	}
	if original.Qdcount != decoded.Qdcount {
		t.Errorf("Qdcount mismatch: original %d, decoded %d", original.Qdcount, decoded.Qdcount)
	}

	// Парсим флаги для проверки
	QR, _, AA, _, RD, RA, _, _ := decoded.parseFlags(&decoded.Flags)

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

	fmt.Printf("Round-trip test passed:\n")
	fmt.Printf("  Original: ID=%04x, Flags=%04x\n", original.ID, original.Flags)
	fmt.Printf("  Decoded:  ID=%04x, Flags=%04x\n", decoded.ID, decoded.Flags)
	fmt.Printf("  Flags: QR=%d, AA=%d, RD=%d, RA=%d\n", QR, AA, RD, RA)
}

func TestParseFlagsFromRealDNS(t *testing.T) {
	// Пример реального DNS пакета (заголовок)
	// ID: 0x1234, Flags: 0x8180 (QR=1, RD=1, RA=1)
	dnsPacket := []byte{
		0x12, 0x34, // ID
		0x81, 0x80, // Flags = 0x8180 (QR=1, RD=1, RA=1)
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x02, // ANCOUNT = 2
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x01, // ARCOUNT = 1
		// ... остальная часть пакета
	}

	h, err := HandleHeader(dnsPacket)
	if err != nil {
		t.Fatalf("HandleHeader failed: %v", err)
	}

	if h.ID != 0x1234 {
		t.Errorf("ID incorrect: expected 0x1234, got 0x%04x", h.ID)
	}

	// Проверяем флаги
	QR, _, _, _, RD, RA, _, _ := h.parseFlags(&h.Flags)

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

	fmt.Printf("Real DNS packet test:\n")
	fmt.Printf("  Flags: 0x%04x = %016b\n", h.Flags, h.Flags)
	fmt.Printf("  QR=%d, RD=%d, RA=%d\n", QR, RD, RA)
}

func TestErrorConditions(t *testing.T) {
	// Тест на слишком короткий заголовок
	shortData := []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03} // 6 байт вместо 12
	_, err := HandleHeader(shortData)
	if err == nil {
		t.Error("Expected error for short header, got nil")
	} else if err.Error() != "DNS message is too short" {
		t.Errorf("Expected 'DNS message is too short', got: %v", err)
	}

	// Тест на QDCOUNT = 0
	zeroQdcount := make([]byte, 12)
	binary.BigEndian.PutUint16(zeroQdcount[4:6], 0) // QDCOUNT = 0
	_, err = HandleHeader(zeroQdcount)
	if err == nil {
		t.Error("Expected error for QDCOUNT=0, got nil")
	} else if err.Error() != "QDCOUNT is 0" {
		t.Errorf("Expected 'QDCOUNT is 0', got: %v", err)
	}
}

// Benchmark тест для проверки производительности
func BenchmarkSetFlags(b *testing.B) {
	h := Header{}
	var flags uint16

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.SetFlags(&flags, 1, 0, 0, 0, 1, 1, 0, 0)
	}
}

func BenchmarkParseFlags(b *testing.B) {
	h := Header{}
	var flags uint16 = 0x8180 // QR=1, RD=1, RA=1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.parseFlags(&flags)
	}
}
