package segment

import (
	"encoding/binary"
	"math"

	"github.com/bearlytools/claw/clawc/languages/go/field"
)

// Number is a constraint for numeric types that can be stored in Claw.
type Number interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// SetBool sets a boolean field. Zero values are omitted (sparse encoding).
func SetBool(s *Struct, fieldNum uint16, value bool) {
	if !value {
		// Sparse encoding: remove zero-value fields
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	// Bool stored in Final40 as 1
	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTBool, 1)

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: []byte{1}})
	}
}

// SetInt8 sets an int8 field.
func SetInt8(s *Struct, fieldNum uint16, value int8) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTInt8, uint64(uint8(value)))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: []byte{byte(value)}})
	}
}

// SetInt16 sets an int16 field.
func SetInt16(s *Struct, fieldNum uint16, value int16) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTInt16, uint64(uint16(value)))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeInt16(value)})
	}
}

// SetInt32 sets an int32 field.
func SetInt32(s *Struct, fieldNum uint16, value int32) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTInt32, uint64(uint32(value)))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeInt32(value)})
	}
}

// SetInt64 sets an int64 field. 64-bit values need header + 8 bytes data.
func SetInt64(s *Struct, fieldNum uint16, value int64) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	// 64-bit values: header + 8 bytes of data
	data := make([]byte, HeaderSize+8)
	EncodeHeader(data[0:8], fieldNum, field.FTInt64, 0) // Final40 unused for 64-bit
	binary.LittleEndian.PutUint64(data[8:16], uint64(value))

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeInt64(value)})
	}
}

// SetUint8 sets a uint8 field.
func SetUint8(s *Struct, fieldNum uint16, value uint8) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTUint8, uint64(value))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: []byte{value}})
	}
}

// SetUint16 sets a uint16 field.
func SetUint16(s *Struct, fieldNum uint16, value uint16) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTUint16, uint64(value))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeUint16(value)})
	}
}

// SetUint32 sets a uint32 field.
func SetUint32(s *Struct, fieldNum uint16, value uint32) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTUint32, uint64(value))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeUint32(value)})
	}
}

// SetUint64 sets a uint64 field. 64-bit values need header + 8 bytes data.
func SetUint64(s *Struct, fieldNum uint16, value uint64) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	// 64-bit values: header + 8 bytes of data
	data := make([]byte, HeaderSize+8)
	EncodeHeader(data[0:8], fieldNum, field.FTUint64, 0) // Final40 unused for 64-bit
	binary.LittleEndian.PutUint64(data[8:16], value)

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeUint64(value)})
	}
}

// SetFloat32 sets a float32 field.
func SetFloat32(s *Struct, fieldNum uint16, value float32) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	// Float32 bits stored in Final40
	bits := math.Float32bits(value)
	hdr := make([]byte, HeaderSize)
	EncodeHeader(hdr, fieldNum, field.FTFloat32, uint64(bits))

	s.insertField(fieldNum, hdr)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeFloat32(value)})
	}
}

// SetFloat64 sets a float64 field. 64-bit values need header + 8 bytes data.
func SetFloat64(s *Struct, fieldNum uint16, value float64) {
	if value == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return
	}

	// 64-bit values: header + 8 bytes of data
	bits := math.Float64bits(value)
	data := make([]byte, HeaderSize+8)
	EncodeHeader(data[0:8], fieldNum, field.FTFloat64, 0) // Final40 unused for 64-bit
	binary.LittleEndian.PutUint64(data[8:16], bits)

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: EncodeFloat64(value)})
	}
}

// SetEnum sets an enum field (stored as uint16).
func SetEnum(s *Struct, fieldNum uint16, value uint16) {
	// Enums are stored like uint16 but with their own field type
	// For now, use the same encoding as uint16
	SetUint16(s, fieldNum, value)
}

// GetBool gets a boolean field value.
func GetBool(s *Struct, fieldNum uint16) bool {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return false
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return final40 != 0
}

// GetInt8 gets an int8 field value.
func GetInt8(s *Struct, fieldNum uint16) int8 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return int8(final40)
}

// GetInt16 gets an int16 field value.
func GetInt16(s *Struct, fieldNum uint16) int16 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return int16(final40)
}

// GetInt32 gets an int32 field value.
func GetInt32(s *Struct, fieldNum uint16) int32 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return int32(final40)
}

// GetInt64 gets an int64 field value.
func GetInt64(s *Struct, fieldNum uint16) int64 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	// 64-bit value is stored after the header
	return int64(binary.LittleEndian.Uint64(s.seg.data[offset+HeaderSize:]))
}

// GetUint8 gets a uint8 field value.
func GetUint8(s *Struct, fieldNum uint16) uint8 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return uint8(final40)
}

// GetUint16 gets a uint16 field value.
func GetUint16(s *Struct, fieldNum uint16) uint16 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return uint16(final40)
}

// GetUint32 gets a uint32 field value.
func GetUint32(s *Struct, fieldNum uint16) uint32 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return uint32(final40)
}

// GetUint64 gets a uint64 field value.
func GetUint64(s *Struct, fieldNum uint16) uint64 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	// 64-bit value is stored after the header
	return binary.LittleEndian.Uint64(s.seg.data[offset+HeaderSize:])
}

// GetFloat32 gets a float32 field value.
func GetFloat32(s *Struct, fieldNum uint16) float32 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	return math.Float32frombits(uint32(final40))
}

// GetFloat64 gets a float64 field value.
func GetFloat64(s *Struct, fieldNum uint16) float64 {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return 0
	}
	// 64-bit value is stored after the header
	bits := binary.LittleEndian.Uint64(s.seg.data[offset+HeaderSize:])
	return math.Float64frombits(bits)
}
