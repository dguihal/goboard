package utils

import (
	"encoding/binary"
)

// IToB returns an 8-byte big endian representation of v.
func IToB(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
