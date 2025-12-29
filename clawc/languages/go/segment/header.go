package segment

import (
	"encoding/binary"

	"github.com/bearlytools/claw/clawc/languages/go/field"
)

// Header layout (8 bytes, little-endian):
//   Bytes 0-1: Field number (uint16)
//   Byte 2:    Field type (uint8)
//   Bytes 3-7: Final40 (40 bits) - value or size depending on type

const (
	HeaderSize   = 8
	MaxFinal40   = 1<<40 - 1 // Maximum value for 40-bit field
	final40Shift = 24        // Bit position where Final40 starts
)

// EncodeHeader writes a complete 8-byte header to the given buffer.
// The buffer must be at least 8 bytes.
func EncodeHeader(buf []byte, fieldNum uint16, fieldType field.Type, final40 uint64) {
	if len(buf) < HeaderSize {
		panic("segment: header buffer too small")
	}
	if final40 > MaxFinal40 {
		panic("segment: final40 value exceeds 40 bits")
	}

	// Field number (bytes 0-1)
	binary.LittleEndian.PutUint16(buf[0:2], fieldNum)

	// Field type (byte 2)
	buf[2] = byte(fieldType)

	// Final40 (bytes 3-7) - pack into bits 24-63 of a uint64
	// First clear bytes 3-7
	buf[3] = 0
	buf[4] = 0
	buf[5] = 0
	buf[6] = 0
	buf[7] = 0

	// Read the full uint64, set the final40 bits, write back
	u := binary.LittleEndian.Uint64(buf)
	u |= final40 << final40Shift
	binary.LittleEndian.PutUint64(buf, u)
}

// DecodeHeader reads header fields from an 8-byte buffer.
func DecodeHeader(buf []byte) (fieldNum uint16, fieldType field.Type, final40 uint64) {
	if len(buf) < HeaderSize {
		panic("segment: header buffer too small")
	}

	fieldNum = binary.LittleEndian.Uint16(buf[0:2])
	fieldType = field.Type(buf[2])

	u := binary.LittleEndian.Uint64(buf)
	final40 = u >> final40Shift

	return
}

// EncodeHeaderFieldNum updates just the field number in a header.
func EncodeHeaderFieldNum(buf []byte, fieldNum uint16) {
	binary.LittleEndian.PutUint16(buf[0:2], fieldNum)
}

// EncodeHeaderFieldType updates just the field type in a header.
func EncodeHeaderFieldType(buf []byte, fieldType field.Type) {
	buf[2] = byte(fieldType)
}

// EncodeHeaderFinal40 updates just the Final40 value in a header.
func EncodeHeaderFinal40(buf []byte, final40 uint64) {
	if final40 > MaxFinal40 {
		panic("segment: final40 value exceeds 40 bits")
	}

	// Clear bytes 3-7 first
	buf[3] = 0
	buf[4] = 0
	buf[5] = 0
	buf[6] = 0
	buf[7] = 0

	// Read the full uint64, set the final40 bits, write back
	u := binary.LittleEndian.Uint64(buf)
	u |= final40 << final40Shift
	binary.LittleEndian.PutUint64(buf, u)
}

// DecodeHeaderFinal40 reads just the Final40 value from a header.
func DecodeHeaderFinal40(buf []byte) uint64 {
	u := binary.LittleEndian.Uint64(buf)
	return u >> final40Shift
}

// MakeHeader creates an 8-byte header as a new slice.
func MakeHeader(fieldNum uint16, fieldType field.Type, final40 uint64) []byte {
	buf := make([]byte, HeaderSize)
	EncodeHeader(buf, fieldNum, fieldType, final40)
	return buf
}

// EncodeScalarInHeader encodes a scalar value that fits in Final40 (values < 40 bits).
// This is used for bool, int8-32, uint8-32, float32.
func EncodeScalarInHeader(buf []byte, fieldNum uint16, fieldType field.Type, value uint64) {
	EncodeHeader(buf, fieldNum, fieldType, value)
}
