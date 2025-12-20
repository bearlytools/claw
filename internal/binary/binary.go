// Package binary replaces the encoding/binary package in the standard library for little endian encoding using generics.
package binary

import (
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/exp/constraints"
)

// Enc is the little-endian binary encoder. Do not change this.
var Enc = binary.LittleEndian

// Get gets any Uint size from a []byte slice.
func Get[T constraints.Integer](b []byte) T {
	_ = b[len(b)-1] // bounds check hint to compiler; see golang.org/issue/14808

	var r T // This is only used for type detction.
	switch any(r).(type) {
	case int8:
		return T(int8(b[0]))
	case int16:
		return T(int16(uint16(b[0]) | uint16(b[1])<<8))
	case int32:
		return T(int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24))
	case int64:
		return T(int64(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56))
	case uint8:
		return T(uint8(b[0]))
	case uint16:
		return T(uint16(b[0]) | uint16(b[1])<<8)
	case uint32:
		return T(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
	case uint64:
		return T(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
	}
	panic(fmt.Sprintf("unsupported type that passed the type constraint %T", r))
}

// GetBuffer reads from an io.Reader and decodes into the specified integer type.
func GetBuffer[T constraints.Integer](r io.Reader) (T, error) {
	var rt T // This is only used for type detction.
	switch any(rt).(type) {
	case int8:
		var b [1]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(int8(b[0])), nil
	case int16:
		var b [2]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(int16(uint16(b[0]) | uint16(b[1])<<8)), nil
	case int32:
		var b [4]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)), nil
	case int64:
		var b [8]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(int64(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)), nil
	case uint8:
		var b [1]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(uint8(b[0])), nil
	case uint16:
		var b [2]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(uint16(b[0]) | uint16(b[1])<<8), nil
	case uint32:
		var b [4]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24), nil
	case uint64:
		var b [8]byte
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			return 0, err
		}
		return T(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56), nil
	}
	panic(fmt.Sprintf("unsupported type that passed the type constraint %T", r))
}

// Put puts any Uint size into a []byte slice.
func Put[T constraints.Integer](b []byte, v T) {
	switch any(v).(type) {
	case uint8:
		b[0] = byte(v)
		return
	case uint16:
		binary.LittleEndian.PutUint16(b, uint16(v))
		return
	case uint32:
		binary.LittleEndian.PutUint32(b, uint32(v))
		return
	}
	binary.LittleEndian.PutUint64(b, uint64(v))
}

// PutBuffer encodes an integer into the passed Buffer.
func PutBuffer[T constraints.Integer](buff io.Writer, v T) error {
	var b []byte
	switch any(v).(type) {
	case int8:
		v = T(uint8(v))
		b = make([]byte, 1)
	case int16:
		v = T(uint16(v))
		b = make([]byte, 2)
	case int32:
		v = T(uint32(v))
		b = make([]byte, 4)
	case int64:
		v = T(uint64(v))
		b = make([]byte, 8)
	}

	Put(b, v)
	_, err := buff.Write(b)
	return err
}

// Direct type-specific functions below avoid type switch overhead for hot paths.

// GetInt8 reads an int8 from a byte slice.
func GetInt8(b []byte) int8 {
	return int8(b[0])
}

// GetInt16 reads an int16 from a byte slice (little-endian).
func GetInt16(b []byte) int16 {
	return int16(uint16(b[0]) | uint16(b[1])<<8)
}

// GetInt32 reads an int32 from a byte slice (little-endian).
func GetInt32(b []byte) int32 {
	return int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}

// GetInt64 reads an int64 from a byte slice (little-endian).
func GetInt64(b []byte) int64 {
	return int64(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
}

// GetUint8 reads a uint8 from a byte slice.
func GetUint8(b []byte) uint8 {
	return b[0]
}

// GetUint16 reads a uint16 from a byte slice (little-endian).
func GetUint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

// GetUint32 reads a uint32 from a byte slice (little-endian).
func GetUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// GetUint64 reads a uint64 from a byte slice (little-endian).
func GetUint64(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// PutInt8 writes an int8 to a byte slice.
func PutInt8(b []byte, v int8) {
	b[0] = byte(v)
}

// PutInt16 writes an int16 to a byte slice (little-endian).
func PutInt16(b []byte, v int16) {
	binary.LittleEndian.PutUint16(b, uint16(v))
}

// PutInt32 writes an int32 to a byte slice (little-endian).
func PutInt32(b []byte, v int32) {
	binary.LittleEndian.PutUint32(b, uint32(v))
}

// PutInt64 writes an int64 to a byte slice (little-endian).
func PutInt64(b []byte, v int64) {
	binary.LittleEndian.PutUint64(b, uint64(v))
}

// PutUint8 writes a uint8 to a byte slice.
func PutUint8(b []byte, v uint8) {
	b[0] = v
}

// PutUint16 writes a uint16 to a byte slice (little-endian).
func PutUint16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

// PutUint32 writes a uint32 to a byte slice (little-endian).
func PutUint32(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

// PutUint64 writes a uint64 to a byte slice (little-endian).
func PutUint64(b []byte, v uint64) {
	binary.LittleEndian.PutUint64(b, v)
}

// GetBufferInt8 reads an int8 from an io.Reader.
func GetBufferInt8(r io.Reader) (int8, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return int8(b[0]), nil
}

// GetBufferInt16 reads an int16 from an io.Reader (little-endian).
func GetBufferInt16(r io.Reader) (int16, error) {
	var b [2]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return int16(uint16(b[0]) | uint16(b[1])<<8), nil
}

// GetBufferInt32 reads an int32 from an io.Reader (little-endian).
func GetBufferInt32(r io.Reader) (int32, error) {
	var b [4]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24), nil
}

// GetBufferInt64 reads an int64 from an io.Reader (little-endian).
func GetBufferInt64(r io.Reader) (int64, error) {
	var b [8]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return int64(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56), nil
}

// GetBufferUint8 reads a uint8 from an io.Reader.
func GetBufferUint8(r io.Reader) (uint8, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// GetBufferUint16 reads a uint16 from an io.Reader (little-endian).
func GetBufferUint16(r io.Reader) (uint16, error) {
	var b [2]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return uint16(b[0]) | uint16(b[1])<<8, nil
}

// GetBufferUint32 reads a uint32 from an io.Reader (little-endian).
func GetBufferUint32(r io.Reader) (uint32, error) {
	var b [4]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}

// GetBufferUint64 reads a uint64 from an io.Reader (little-endian).
func GetBufferUint64(r io.Reader) (uint64, error) {
	var b [8]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56, nil
}

// PutBufferInt8 writes an int8 to an io.Writer.
func PutBufferInt8(w io.Writer, v int8) error {
	_, err := w.Write([]byte{byte(v)})
	return err
}

// PutBufferInt16 writes an int16 to an io.Writer (little-endian).
func PutBufferInt16(w io.Writer, v int16) error {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(v))
	_, err := w.Write(b[:])
	return err
}

// PutBufferInt32 writes an int32 to an io.Writer (little-endian).
func PutBufferInt32(w io.Writer, v int32) error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	_, err := w.Write(b[:])
	return err
}

// PutBufferInt64 writes an int64 to an io.Writer (little-endian).
func PutBufferInt64(w io.Writer, v int64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	_, err := w.Write(b[:])
	return err
}

// PutBufferUint8 writes a uint8 to an io.Writer.
func PutBufferUint8(w io.Writer, v uint8) error {
	_, err := w.Write([]byte{v})
	return err
}

// PutBufferUint16 writes a uint16 to an io.Writer (little-endian).
func PutBufferUint16(w io.Writer, v uint16) error {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// PutBufferUint32 writes a uint32 to an io.Writer (little-endian).
func PutBufferUint32(w io.Writer, v uint32) error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// PutBufferUint64 writes a uint64 to an io.Writer (little-endian).
func PutBufferUint64(w io.Writer, v uint64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	_, err := w.Write(b[:])
	return err
}
