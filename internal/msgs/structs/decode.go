package structs

import (
	"bytes"
	"fmt"
	"io"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
)

func (s *Struct) unmarshal(r io.Reader) error {
	h := GenericHeader(make([]byte, 8))
	n, err := r.Read(h)
	if n != 8 {
		return fmt.Errorf("could only read %d bytes, a Struct header is always 8 bytes", n)
	}
	if err != nil {
		return err
	}

	ft := field.Type(h.Next8())
	if ft != field.FTStruct {
		return fmt.Errorf("expecting Struct, got %v", ft)
	}

	size := h.Final40()
	if size%8 != 0 {
		return fmt.Errorf("Struct malformed: must have a size divisible by 8, was %d", h.Final40())
	}

	buffer := make([]byte, size)

	_, err = r.Read(buffer)
	if err != nil {
		return fmt.Errorf("problem reading Struct data: %w", err)
	}

	return s.unmarshalFields(&buffer, 0)
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
			s.excess = *buffer
			addToTotal(s, 8+len(s.excess))
			return nil
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
			err = s.decodeStruct(buffer, fieldNum)
		case field.FTListBool:
			err = s.decodeListBool(buffer, fieldNum)
		case field.FTList8, field.FTList16, field.FTList32, field.FTList64:
			err = s.decodeListNumber(buffer, fieldNum)
		case field.FTListBytes:
			err = s.decodeListBytes(buffer, fieldNum)
		case field.FTListStruct:
			err = s.decodeListStruct(buffer, fieldNum)
		default:
			err = fmt.Errorf("got field type %v that we don't support", fieldType)
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
	f := s.fields[fieldNum-1]
	f.header = (*buffer)[0:8]
	s.fields[fieldNum-1] = f
	addToTotal(s, 8)
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
		f := s.fields[fieldNum-1]
		f.header = (*buffer)[:8]
		addToTotal(s, 8)
		*buffer = (*buffer)[8:]
	case 64:
		if len(*buffer) < 16 {
			return fmt.Errorf("can't decode a 64 bit number with < 128 bits")
		}
		f := s.fields[fieldNum-1]
		f.header = (*buffer)[:8]
		v := (*buffer)[8:16]
		f.ptr = unsafe.Pointer(&v)
		s.fields[fieldNum] = f
		addToTotal(s, 16)
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
	f := s.fields[fieldNum-1]
	f.header = (*buffer)[:8]
	b := (*buffer)[8:withPadding]
	f.ptr = unsafe.Pointer(&b)
	s.fields[fieldNum-1] = f
	addToTotal(s, withPadding)
	*buffer = (*buffer)[withPadding:]
	return nil
}

func (s *Struct) decodeListBool(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum-1]
	f.header = (*buffer)[:8]

	ptr, err := NewBoolFromBytes(buffer, s) // This handles our additions to s.total.
	if err != nil {
		return err
	}
	f.ptr = unsafe.Pointer(ptr)
	s.fields[fieldNum-1] = f
	return nil
}

func (s *Struct) decodeListNumber(buffer *[]byte, fieldNum uint16) error {
	m := s.mapping[int(fieldNum-1)]
	f := s.fields[fieldNum-1]
	f.header = (*buffer)[:8]

	var uptr unsafe.Pointer
	switch m.ListType {
	case field.FTInt8:
		ptr, err := NewNumberFromBytes[int8](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTInt16:
		ptr, err := NewNumberFromBytes[int16](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTInt32:
		ptr, err := NewNumberFromBytes[int32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTInt64:
		ptr, err := NewNumberFromBytes[int64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint8:
		ptr, err := NewNumberFromBytes[uint8](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint16:
		ptr, err := NewNumberFromBytes[uint16](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint32:
		ptr, err := NewNumberFromBytes[uint32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTUint64:
		ptr, err := NewNumberFromBytes[uint64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTFloat32:
		ptr, err := NewNumberFromBytes[float32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTFloat64:
		ptr, err := NewNumberFromBytes[float64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	default:
		panic(fmt.Sprintf("Struct.decodeListNumber() called with field that is mapped to value with type: %v", m.ListType))
	}
	f.ptr = uptr
	s.fields[fieldNum-1] = f
	return nil
}

func (s *Struct) decodeListBytes(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum-1]
	f.header = (*buffer)[:8]

	ptr, err := NewBytesFromBytes(buffer, s)
	if err != nil {
		return err
	}
	f.ptr = unsafe.Pointer(ptr)
	s.fields[fieldNum-1] = f
	return nil
}

func (s *Struct) decodeStruct(buffer *[]byte, fieldNum uint16) error {
	// We need the mapping for the sub Struct.
	m := s.mapping[fieldNum-1].Mapping
	if m == nil {
		return fmt.Errorf("received a fieldNum(%d) with a type that Says it is a Struct, but it is a %v", fieldNum, s.mapping[fieldNum-1].Type)
	}

	// Structs use a Reader, so let's give it a reader.
	r := readers.Get().(*bytes.Reader)
	r.Reset(*buffer)
	defer readers.Put(r)

	sub := New(fieldNum, m, s)
	err := sub.unmarshal(r)
	if err != nil {
		return err
	}

	s.fields[fieldNum-1].header = sub.header
	s.fields[fieldNum-1].ptr = unsafe.Pointer(sub)

	*buffer = (*buffer)[*sub.structTotal:]
	return nil
}

func (s *Struct) decodeListStruct(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum-1]
	header := GenericHeader((*buffer)[:8])
	*buffer = (*buffer)[8:]

	numItems := header.Final40()
	sl := make([]*Struct, int(numItems))

	r := readers.Get().(*bytes.Reader)
	for i := 0; i < int(numItems); i++ {
		item := New(fieldNum, s.mapping[fieldNum].Mapping, s)
		r.Reset(*buffer)
		if err := s.unmarshal(r); err != nil {
			return err
		}
		sl[i] = item
		*buffer = (*buffer)[int(item.header.Final40()):]
	}
	f.header = header
	f.ptr = unsafe.Pointer(&sl)
	s.fields[fieldNum-1] = f
	return nil
}
