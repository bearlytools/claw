package structs

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
)

func (s *Struct) unmarshal(r io.Reader, buffer *[]byte) error {
	hb := headerPool.Get().([]byte)

	n, err := r.Read(hb)
	if n != 8 {
		return fmt.Errorf("could only read %d bytes, a Struct header is always 8 bytes", n)
	}
	if err != nil {
		return err
	}

	h := Header{}
	if err := h.Read(hb); err != nil {
		return err
	}
	if err := h.validate(); err != nil {
		return err
	}

	padding := (h.DataSize % 8)
	total := int(h.DataSize + padding)
	if cap(*buffer) < total {
		*buffer = make([]byte, total)
	}
	*buffer = (*buffer)[0:total]

	n, err = r.Read(*buffer)
	if err != nil {
		return fmt.Errorf("Struct data should be %d bytes + %d padding, but could only read %d bytes", h.DataSize, padding, n)
	}

	*buffer = (*buffer)[:h.DataSize]
	s.unmarshalFields(buffer, 0)
	return nil

}

func (s *Struct) unmarshalFields(buffer *[]byte, lastNum uint16) error {
	maxFields := uint16(len(s.mapping))

	var (
		store     uint64
		fieldNum  uint16
		fieldType field.Type
	)

	for {
		if len(*buffer) < 8 {
			return fmt.Errorf("field inside Struct was malformed: not enough room for field number and field type")
		}
		store = binary.Get[uint64]((*buffer)[:8])
		fieldNum = bits.GetValue[uint64, uint16](store, fieldNumMask, 0)
		fieldType = field.Type(bits.GetValue[uint64, uint8](store, fieldTypeMask, 17))

		if fieldNum <= lastNum {
			return fmt.Errorf("Struct was malformed: field %d came after field %d", fieldNum, lastNum)
		}

		// This means we have fields that we don't know about, but are likely from a
		// program writing an updated version of our Struct that has more fields. So we
		// need to retain our data so that even though the user can't see it, we don't
		// drop it.
		if fieldNum > maxFields {
			// If we don't have enough capacity in our slice, extend the slice capacity
			// up to the fieldNum. Remember that while fields must come in order, they
			// can skip field numbers, so we may have to extend multiple times with append.
			if cap(s.fields) < int(fieldNum) {
				s.fields = s.fields[:cap(s.fields)]
				for i := 0; i < int(fieldNum)-cap(s.fields); i++ {
					s.fields = append(s.fields, nil)
				}
			} else {
				// We have enough capacity in our underlying aray, so just extend the slice.
				s.fields = s.fields[:fieldNum]
			}
		}

		var err error
		switch fieldType {
		case field.FTBool:
			err = s.decodeBool(buffer, fieldNum)
		case field.FTInt8:
			err = s.decodeNum(buffer, fieldNum, 8)
		case field.FTInt16:
			err = s.decodeNum(buffer, fieldNum, 16)
		case field.FTInt32:
			err = s.decodeNum(buffer, fieldNum, 32)
		case field.FTInt64:
			err = s.decodeNum(buffer, fieldNum, 64)
		case field.FTUint8:
			err = s.decodeNum(buffer, fieldNum, 8)
		case field.FTUint16:
			err = s.decodeNum(buffer, fieldNum, 16)
		case field.FTUint32:
			err = s.decodeNum(buffer, fieldNum, 32)
		case field.FTUint64:
			err = s.decodeNum(buffer, fieldNum, 64)
		case field.FTFloat32:
			err = s.decodeNum(buffer, fieldNum, 32)
		case field.FTFloat64:
			err = s.decodeNum(buffer, fieldNum, 64)
		case field.FTString, field.FTBytes:
			err = s.decodeBytes(buffer, fieldNum)
		case field.FTStruct:
		case field.FTListBool:
			err = s.decodeListBool(buffer, fieldNum)
		case field.FTList8, field.FTList16, field.FTList32, field.FTList64:
			err = s.decodeListNumber(buffer, fieldNum)
		case field.FTListBytes:
			err = s.decodeListBytes(buffer, fieldNum)
		case field.FTListStruct:
		default:
		}
		if err != nil {
			return err
		}
	}
}

// decodeBool will decode a boolean value from the buffer into .fields[fieldNum] and
// advance the buffer for the next value.
func (s *Struct) decodeBool(buffer *[]byte, fieldNum uint16) error {
	if len(*buffer) < 8 {
		return fmt.Errorf("can't decode bool value, not enough bytes for bool value")
	}
	s.fields[fieldNum] = (*buffer)[0:8]
	*buffer = (*buffer)[8:]
	return nil
}

// decodeNum will decode a number value from the buffer into .fields[fieldNum] and
// advance the buffer for the next value.
func (s *Struct) decodeNum(buffer *[]byte, fieldNum uint16, numSize int8) error {
	switch numSize {
	case 8, 16, 32:
		if len(*buffer) < 8 {
			return fmt.Errorf("can't decode a 8, 16, or 32 bit number with < 64 bits")
		}
		s.fields[fieldNum] = (*buffer)[:8]
		*buffer = (*buffer)[8:]
	case 64:
		if len(*buffer) < 16 {
			return fmt.Errorf("can't decode a 64 bit number with < 128 bits")
		}
		s.fields[fieldNum] = (*buffer)[:8]
		*buffer = (*buffer)[16:]
	default:
		return fmt.Errorf("Struct.decodeNum() numSize was %d, must be 32 or 64", numSize)
	}
	return nil
}

// decodeBytes will decode a bytes/string value from the buffer into .fields[fieldNum] and
// advance the buffer for the next value.
func (s *Struct) decodeBytes(buffer *[]byte, fieldNum uint16) error {
	l := len(*buffer)
	if l < 8 {
		return fmt.Errorf("Struct.decodeBytes() header was < 64 bits")
	}

	i := binary.Get[uint64]((*buffer)[:8])
	size := bits.GetValue[uint64, uint64](i, dataSizeMask, 24)

	withPadding := 8 + size + (size % 8) // header + data + padding
	if l < int(withPadding) {
		return fmt.Errorf("Struct.decodeBytes() found string/byte field that was clipped in size")
	}
	s.fields[fieldNum] = (*buffer)[:withPadding]
	*buffer = (*buffer)[withPadding:]
	return nil
}

func (s *Struct) decodeListBool(buffer *[]byte, fieldNum uint16) error {
	ptr, err := NewBoolFromBytes(buffer, s.total)
	if err != nil {
		return err
	}
	listIndex := s.fieldNumToList[fieldNum]
	s.lists[listIndex] = unsafe.Pointer(ptr)
	return nil
}

func (s *Struct) decodeListNumber(buffer *[]byte, fieldNum uint16) error {
	m := s.mapping[int(fieldNum)]
	listIndex := s.fieldNumToList[fieldNum]
	var uptr unsafe.Pointer
	switch m.ListType.Type {
	case field.FTInt8:
		ptr, err := NewNumberFromBytes[int8](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTInt16:
		ptr, err := NewNumberFromBytes[int16](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTInt32:
		ptr, err := NewNumberFromBytes[int32](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTInt64:
		ptr, err := NewNumberFromBytes[int64](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint8:
		ptr, err := NewNumberFromBytes[uint8](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint16:
		ptr, err := NewNumberFromBytes[uint16](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint32:
		ptr, err := NewNumberFromBytes[uint32](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint64:
		ptr, err := NewNumberFromBytes[uint64](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTFloat32:
		ptr, err := NewNumberFromBytes[float32](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTFloat64:
		ptr, err := NewNumberFromBytes[float64](buffer, s.total)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	default:
		panic(fmt.Sprintf("Struct.decodeListNumber() called with field that is mapped to value with type: %v", m.ListType.Type))
	}
	s.lists[listIndex] = uptr
	return nil
}

func (s *Struct) decodeListBytes(buffer *[]byte, fieldNum uint16) error {
	ptr, err := NewBytesFromBytes(buffer, s.total)
	if err != nil {
		return err
	}
	listIndex := s.fieldNumToList[fieldNum]
	s.lists[listIndex] = unsafe.Pointer(ptr)
	return nil
}
