package patch

import (
	"encoding/binary"
	"math"
)

// Number encoding helpers using little-endian format to match claw wire format.

func encodeInt16(v int16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(v))
	return b
}

func encodeUint16(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

func encodeInt32(v int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	return b
}

func encodeUint32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func encodeInt64(v int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	return b
}

func encodeUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func encodeFloat32(v float32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	return b
}

func encodeFloat64(v float64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(v))
	return b
}

// Decode helpers
func decodeInt16(b []byte) int16 {
	return int16(binary.LittleEndian.Uint16(b))
}

func decodeUint16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

func decodeInt32(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b))
}

func decodeUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func decodeInt64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}

func decodeUint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

func decodeFloat32(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

func decodeFloat64(b []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(b))
}
