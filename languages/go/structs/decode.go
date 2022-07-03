package structs

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
)

func (s *Struct) unmarshal(r io.Reader) (int, error) {
	read := 0
	h := NewGenericHeader()
	read, err := r.Read(h)
	if read != 8 {
		return read, fmt.Errorf("could only read %d bytes, a Struct header is always 8 bytes", read)
	}
	if err != nil {
		return read, err
	}

	ft := field.Type(h.FieldType())
	if ft != field.FTStruct {
		return read, fmt.Errorf("expecting Struct, got %v", ft)
	}

	size := h.Final40()
	if size%8 != 0 {
		return read, fmt.Errorf("Struct malformed: must have a size divisible by 8, was %d", h.Final40())
	}

	log.Println("Struct says it is: ", size)
	buffer := make([]byte, size-8) // -8 because we read the buffer

	if len(buffer) == 0 {
		return read, nil
	}

	n, err := r.Read(buffer)
	read += n
	if err != nil {
		log.Println("this is the buffer size: ", len(buffer))
		return read, fmt.Errorf("problem reading Struct data: %w", err)
	}
	if int(size-8) != n {
		panic(fmt.Sprintf("read %d bytes, expected %d", n, size-8))
	}
	log.Println("struct read ", read)
	err = s.unmarshalFields(&buffer)
	if err != nil {
		return read, err
	}
	st := atomic.LoadInt64(s.structTotal)
	if read != int(st) {
		return read, fmt.Errorf("Struct was %d in length, but only found %d worth of fields", read, st)
	}

	return read, nil
}

func (s *Struct) unmarshalFields(buffer *[]byte) error {
	maxFields := uint16(len(s.mapping))
	defer log.Println("unmarshal() end")

	var (
		fieldNum  uint16
		fieldType field.Type
		lastNum   int32 = -1
	)

	entry := 1
	for len(*buffer) > 0 {
		if len(*buffer) < 8 {
			return fmt.Errorf("field inside Struct was malformed: not enough room for field number and field type")
		}
		log.Println("buffer size: ", len(*buffer))

		h := GenericHeader((*buffer)[:8])
		fieldNum = h.FieldNum()
		fieldType = field.Type(h.FieldType())

		if int32(fieldNum) <= lastNum {
			log.Println(*buffer)
			return fmt.Errorf("Struct was malformed: field %d came after field %d", fieldNum, lastNum)
		}
		lastNum = int32(fieldNum)

		// This means we have fields that we don't know about, but are likely from a
		// program writing an updated version of our Struct that has more fields. So we
		// need to retain our data so that even though the user can't see it, we don't
		// drop it.
		if fieldNum >= maxFields {
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
		case field.FTListBools:
			err = s.decodeListBool(buffer, fieldNum)
		case field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
			field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
			field.FTListFloat32, field.FTListFloat64:
			err = s.decodeListNumber(buffer, fieldNum)
		case field.FTListBytes:
			err = s.decodeListBytes(buffer, fieldNum)
		case field.FTListStructs:
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

	f := s.fields[fieldNum]
	f.header = (*buffer)[0:8]
	s.fields[fieldNum] = f
	addToTotal(s, 8)
	*buffer = (*buffer)[8:]
	return nil
}

// decodeNum will decode a number value from the buffer into .fields[fieldNum] and
// advance the buffer for the next value.
func (s *Struct) decodeNum(buffer *[]byte, fieldNum uint16, numSize int8) error {
	if int(fieldNum) >= len(s.fields) {
		return fmt.Errorf("fieldNum %d doesn't exist", fieldNum)
	}

	switch numSize {
	case 8, 16, 32:
		if len(*buffer) < 8 {
			return fmt.Errorf("can't decode a 8, 16, or 32 bit number with < 64 bits")
		}
		f := s.fields[fieldNum]
		f.header = (*buffer)[:8]
		s.fields[fieldNum] = f
		addToTotal(s, 8)
		*buffer = (*buffer)[8:]
	case 64:
		if len(*buffer) < 16 {
			return fmt.Errorf("can't decode a 64 bit number with < 128 bits")
		}
		f := s.fields[fieldNum]
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
	if size == 0 {
		return fmt.Errorf("Struct.decodeBytes() received a Bytes field of size 0 which is invalid")
	}

	withPadding := SizeWithPadding(size) + 8 // header + data + padding
	if l < int(withPadding) {
		return fmt.Errorf("Struct.decodeBytes() found string/byte field that was clipped in size, got %d, want %d", l, withPadding)
	}
	f := s.fields[fieldNum]

	f.header = (*buffer)[:8]
	b := (*buffer)[8 : 8+size] // from end of header to end of data without padding
	f.ptr = unsafe.Pointer(&b)

	s.fields[fieldNum] = f
	log.Println("addToTotal: ", withPadding)
	addToTotal(s, withPadding)
	*buffer = (*buffer)[withPadding:]
	return nil
}

func (s *Struct) decodeListBool(buffer *[]byte, fieldNum uint16) error {
	h, ptr, err := NewBoolFromBytes(buffer, s) // This handles our additions to s.structTotal
	if err != nil {
		return err
	}

	f := s.fields[fieldNum]
	f.header = h
	f.ptr = unsafe.Pointer(ptr)
	s.fields[fieldNum] = f
	return nil
}

func (s *Struct) decodeListBytes(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum]

	ptr, err := NewBytesFromBytes(buffer, s)
	if err != nil {
		return err
	}
	f.header = ptr.header
	f.ptr = unsafe.Pointer(ptr)
	s.fields[fieldNum] = f
	return nil
}

func (s *Struct) decodeListNumber(buffer *[]byte, fieldNum uint16) error {
	m := s.mapping[int(fieldNum)]
	f := s.fields[fieldNum]
	f.header = (*buffer)[:8]
	log.Println("fieldNum: ", f.header.FieldNum())
	var uptr unsafe.Pointer
	switch m.Type {
	case field.FTListInt8:
		ptr, err := NewNumberFromBytes[int8](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt16:
		ptr, err := NewNumberFromBytes[int16](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt32:
		ptr, err := NewNumberFromBytes[int32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt64:
		ptr, err := NewNumberFromBytes[int64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint8:
		ptr, err := NewNumberFromBytes[uint8](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint16:
		ptr, err := NewNumberFromBytes[uint16](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint32:
		ptr, err := NewNumberFromBytes[uint32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint64:
		ptr, err := NewNumberFromBytes[uint64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListFloat32:
		ptr, err := NewNumberFromBytes[float32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListFloat64:
		ptr, err := NewNumberFromBytes[float64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	default:
		panic(fmt.Sprintf("Struct.decodeListNumber() called with field that is mapped to value with type: %v", m.Type))
	}
	f.ptr = uptr
	s.fields[fieldNum] = f
	return nil
}

func (s *Struct) decodeStruct(buffer *[]byte, fieldNum uint16) error {
	// We need the mapping for the sub Struct.
	m := s.mapping[fieldNum].Mapping
	if m == nil { // This means that the contained Struct is the same mapping as the part.
		m = s.mapping
	}

	// Structs use a Reader, so let's give it a reader.
	r := readers.Get().(*bytes.Reader)
	r.Reset(*buffer)
	defer readers.Put(r)

	sub := New(fieldNum, m)
	n, err := sub.unmarshal(r)
	if err != nil {
		return err
	}
	SetStruct(s, fieldNum, sub)

	*buffer = (*buffer)[n:]
	return nil
}

func (s *Struct) decodeListStruct(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum]
	h := GenericHeader((*buffer)[:8])
	*buffer = (*buffer)[8:] // Move ahead of the list header

	numItems := h.Final40()
	if numItems == 0 {
		return fmt.Errorf("cannot decode a list of Structs with list size of 0: encoding error")
	}

	addToTotal(s, 8) // Add list header size
	log.Println("number of items: ", numItems)
	sl := make([]*Struct, int(numItems))
	r := readers.Get().(*bytes.Reader)
	r.Reset(*buffer)
	totalDataSize := 0 // Size without header

	mapping := s.mapping[fieldNum].Mapping
	if mapping == nil { // Means this Struct is the same as the parent.
		mapping = s.mapping
	}
	for i := 0; i < int(numItems); i++ {
		log.Println("decoding list struct item: ", i)
		item := New(0, mapping)
		item.inList = true
		log.Println("\tlength of buffer before item unmarshal: ", len(*buffer)-totalDataSize)
		n, err := item.unmarshal(r)
		if err != nil {
			panic("this happened: " + err.Error())
			return err
		}
		totalDataSize += n
		log.Printf("\tread Struct item: %d bytes", n)
		addToTotal(s, n)
		sl[i] = item
		log.Println("\tlength of buffer after item unmarshal: ", len(*buffer)-totalDataSize)
	}
	*buffer = (*buffer)[totalDataSize:]
	f.header = h
	f.ptr = unsafe.Pointer(&sl)
	s.fields[fieldNum] = f
	return nil
}
