package utils

import (
	"encoding/binary"
	"testing"
)

func TestIToB_RoundTrip(t *testing.T) {
	cases := []uint64{0, 1, 255, 1024, ^uint64(0)}
	for _, v := range cases {
		b := IToB(v)
		if len(b) != 8 {
			t.Fatalf("IToB(%d): expected 8 bytes, got %d", v, len(b))
		}
		got := binary.BigEndian.Uint64(b)
		if got != v {
			t.Errorf("IToB(%d) round-trip failed: got %d", v, got)
		}
	}
}

func TestIToB_BigEndianOrder(t *testing.T) {
	b := IToB(1)
	// Value 1 in big-endian 8 bytes: [0 0 0 0 0 0 0 1]
	if b[7] != 1 {
		t.Errorf("expected last byte to be 1 for value 1, got %d", b[7])
	}
	for i := 0; i < 7; i++ {
		if b[i] != 0 {
			t.Errorf("expected byte %d to be 0, got %d", i, b[i])
		}
	}
}
