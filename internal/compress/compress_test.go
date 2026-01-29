package compress

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"
)

func TestNewCompress(t *testing.T) {
	cmp := NewCompress()
	if cmp == nil {
		t.Fatal("NewCompress() returned nil")
	}
	if cmp.names == nil {
		t.Error("names map is not initialized")
	}
}

func TestAddName(t *testing.T) {
	cmp := NewCompress()
	name := "example.com"
	offset := 42

	cmp.AddName(name, offset)

	info, ok := cmp.names[name]
	if !ok {
		t.Fatal("name not added to the map")
	}

	if info.offset != offset {
		t.Errorf("expected offset %d, got %d", offset, info.offset)
	}

	if info.pointer != 0 {
		t.Errorf("expected pointer 0, got %d", info.pointer)
	}

	cmp.AddName(name, offset+1)
	info, ok = cmp.names[name]
	if !ok {
		t.Fatal("name disappeared from the map")
	}
	if info.offset != offset {
		t.Errorf("offset should not change, expected %d, got %d", offset, info.offset)
	}
}

func TestEncodeName_NewName(t *testing.T) {
	cmp := NewCompress()
	name := "example.com"
	currOffset := 100

	encoded := cmp.EncodeName(name, currOffset)

	expected := []byte{7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}

	if !bytes.Equal(encoded, expected) {
		t.Errorf("expected %v, got %v", expected, encoded)
	}

	info, ok := cmp.names[name]
	if !ok {
		t.Fatal("name not added to the map")
	}
	if info.offset != currOffset {
		t.Errorf("expected offset %d, got %d", currOffset, info.offset)
	}
}

func TestEncodeName_ExistingName(t *testing.T) {
	cmp := NewCompress()
	name := "example.com"
	initialOffset := 50
	currOffset := 100

	// First add the name
	cmp.AddName(name, initialOffset)

	// Then encode it with a later offset
	encoded := cmp.EncodeName(name, currOffset)

	// Should return a pointer (0xC000 | initialOffset)
	expected := make([]byte, 2)
	binary.BigEndian.PutUint16(expected, uint16(0xC000|initialOffset))

	if !bytes.Equal(encoded, expected) {
		t.Errorf("expected %v, got %v", expected, encoded)
	}

	// Verify the pointer was incremented
	info, ok := cmp.names[name]
	if !ok {
		t.Fatal("name disappeared from the map")
	}
	if info.pointer != 1 {
		t.Errorf("expected pointer 1, got %d", info.pointer)
	}
}

func TestEncodeName_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		offset      int
		expected    []byte
		description string
	}{
		{
			name:        "empty name",
			input:       "",
			offset:      0,
			expected:    []byte{0},
			description: "should encode as single null byte",
		},
		{
			name:        "root domain",
			input:       ".",
			offset:      0,
			expected:    []byte{0},
			description: "should encode as single null byte",
		},
		{
			name:        "single label",
			input:       "test",
			offset:      0,
			expected:    []byte{4, 't', 'e', 's', 't', 0},
			description: "should encode as length + label + null",
		},
		{
			name:        "multiple labels",
			input:       "sub.domain.example",
			offset:      0,
			expected:    []byte{3, 's', 'u', 'b', 6, 'd', 'o', 'm', 'a', 'i', 'n', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 0},
			description: "should encode multiple labels correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmp := NewCompress()
			encoded := cmp.EncodeName(tt.input, tt.offset)
			if !bytes.Equal(encoded, tt.expected) {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, encoded)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	cmp := NewCompress()
	name := "example.com"
	offset := 50

	cmp.AddName(name, offset)

	var wg sync.WaitGroup
	n := 100

	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < n; j++ {

				if j%2 == 0 {
					cmp.AddName(name, offset+j)
				} else {
					_ = cmp.EncodeName(name, offset+j+1)
				}
			}
		}()
	}

	wg.Wait()

	cmp.mu.Lock()
	defer cmp.mu.Unlock()

	info, ok := cmp.names[name]
	if !ok {
		t.Fatal("name not in map after concurrent access")
	}

	if info.pointer == 0 {
		t.Errorf("pointer was never incremented, got %d", info.pointer)
	}
}

// add

func TestEncodeName_PointerLogic(t *testing.T) {
	cmp := NewCompress()
	name := "test.com"
	offset1 := 50
	offset2 := 100

	// Первое добавление
	cmp.AddName(name, offset1)

	// Кодируем с offset2 > offset1 - должен вернуть указатель
	encoded := cmp.EncodeName(name, offset2)

	if len(encoded) != 2 {
		t.Errorf("expected pointer (2 bytes), got %d bytes", len(encoded))
	}

	// Проверяем, что это действительно указатель
	pointer := binary.BigEndian.Uint16(encoded)
	if (pointer & 0xC000) != 0xC000 {
		t.Errorf("expected compression pointer, got %04x", pointer)
	}

	// Проверяем счетчик использования
	count := cmp.GetPointerCount(name)
	if count != 1 { // 1 от EncodeName
		t.Errorf("expected pointer count 1, got %d", count)
	}
}

func TestConcurrentAddAndEncode(t *testing.T) {
	cmp := NewCompress()
	var wg sync.WaitGroup

	names := []string{"example.com", "google.com", "github.com"}

	// Добавляем имена
	for i, name := range names {
		wg.Add(1)
		go func(name string, offset int) {
			defer wg.Done()
			cmp.AddName(name, offset)
		}(name, i*100)
	}

	wg.Wait()

	// Параллельно кодируем
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			for _, name := range names {
				_ = cmp.EncodeName(name, iter*1000)
			}
		}(i)
	}

	wg.Wait()

	// Проверяем, что все имена существуют
	for _, name := range names {
		if _, exists := cmp.GetOffset(name); !exists {
			t.Errorf("name %s not found after concurrent operations", name)
		}
	}
}
