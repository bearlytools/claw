package structs

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
)

func (s *Struct) unmarshal(r io.Reader) (int, error) {
	read := 0
	h := GenericHeader(make([]byte, 8))
	read, err := r.Read(h)
	if read != 8 {
		return read, fmt.Errorf("could only read %d bytes, a Struct header is always 8 bytes", read)
	}
	if err != nil {
		return read, err
	}

	ft := field.Type(h.Next8())
	if ft != field.FTStruct {
		return read, fmt.Errorf("expecting Struct, got %v", ft)
	}

	size := h.Final40()
	if size%8 != 0 {
		return read, fmt.Errorf("Struct malformed: must have a size divisible by 8, was %d", h.Final40())
	}

	log.Println("Struct says it is: ", size)
	buffer := make([]byte, size-8) // -8 because we read the buffer

	n, err := r.Read(buffer)
	read += n
	if err != nil {
		return read, fmt.Errorf("problem reading Struct data: %w", err)
	}
	if int(size-8) != n {
		panic(fmt.Sprintf("read %d bytes, expected %d", n, size-8))
	}
	log.Println("struct read ", read)

	return read, s.unmarshalFields(&buffer)
}

func (s *Struct) unmarshalFields(buffer *[]byte) error {
	maxFields := uint16(len(s.mapping))
	defer log.Println("unmarshal() end")

	var (
		fieldNum  uint16
		fieldType field.Type
		lastNum   uint16
	)

	entry := 1
	for len(*buffer) > 0 {
		if len(*buffer) < 8 {
			return fmt.Errorf("field inside Struct was malformed: not enough room for field number and field type")
		}
		log.Println("buffer size: ", len(*buffer))

		h := GenericHeader((*buffer)[:8])
		fieldNum = h.First16()
		fieldType = field.Type(h.Next8())

		if fieldNum <= lastNum {
			log.Println(*buffer)
			return fmt.Errorf("Struct was malformed: field %d came after field %d", fieldNum, lastNum)
		}
		lastNum = fieldNum

		// This means we have fields that we don't know about, but are likely from a
		// program writing an updated version of our Struct that has more fields. So we
		// need to retain our data so that even though the user can't see it, we don't
		// drop it.
		if fieldNum > maxFields {
			log.Printf("wtf: fieldNum %d maxFields %d", fieldNum, maxFields)
			s.excess = *buffer
			addToTotal(s, len(s.excess))
			return nil
		}
		log.Printf("decode field %d/%d", entry, maxFields)
		log.Println("decode fieldNum: ", fieldNum)
		log.Printf("decode fieldType: %v", fieldType)

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
		log.Printf("finished decoding %d/%d", entry, maxFields)
		entry++
	}
	return nil
}

// decodeBool will decode a boolean value from the buffer into .fields[fieldNum] and
// advance the buffer for the next value.
func (s *Struct) decodeBool(buffer *[]byte, fieldNum uint16) error {
	if len(*buffer) < 8 {
		return fmt.Errorf("can't decode bool value, not enough bytes for bool value")
	}
	if int(fieldNum) > len(s.fields) {
		return fmt.Errorf("fieldNum %d doesn't exist", fieldNum)
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
		s.fields[fieldNum-1] = f
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
		s.fields[fieldNum-1] = f
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

	withPadding := 8 + size + PaddingNeeded(8+size) // header + data + padding
	if l < int(withPadding) {
		return fmt.Errorf("Struct.decodeBytes() found string/byte field that was clipped in size")
	}
	f := s.fields[fieldNum-1]

	f.header = (*buffer)[:8]
	b := (*buffer)[8:withPadding] // from end of header to end of data with padding
	f.ptr = unsafe.Pointer(&b)

	s.fields[fieldNum-1] = f
	addToTotal(s, withPadding)
	*buffer = (*buffer)[withPadding:]
	return nil
}

func (s *Struct) decodeListBool(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum-1]
	f.header = (*buffer)[:8]

	ptr, err := NewBoolFromBytes(buffer, s) // This handles our additions to s.structTotal
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
	n, err := sub.unmarshal(r)
	if err != nil {
		return err
	}

	s.fields[fieldNum-1].header = sub.header
	s.fields[fieldNum-1].ptr = unsafe.Pointer(sub)

	*buffer = (*buffer)[n:]
	return nil
}

func (s *Struct) decodeListStruct(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum-1]
	h := GenericHeader((*buffer)[:8])
	*buffer = (*buffer)[8:] // Move ahead of the list header

	numItems := h.Final40()
	sl := make([]*Struct, int(numItems))
	r := readers.Get().(*bytes.Reader)
	for i := 0; i < int(numItems); i++ {
		log.Println("decoding list struct item: ", i)
		item := New(0, s.mapping[fieldNum-1].Mapping, s)
		item.inList = true
		r.Reset(*buffer)
		n, err := item.unmarshal(r)
		if err != nil {
			return err
		}
		sl[i] = item
		*buffer = (*buffer)[n:]
	}
	f.header = h
	f.ptr = unsafe.Pointer(&sl)
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
