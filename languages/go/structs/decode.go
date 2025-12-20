package structs

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/structs/header"
	"github.com/gostdlib/base/context"
)

var dataSizeMask = bits.Mask[uint64](24, 64)

// scanFieldOffsets scans through raw field data and builds an offset index
// without actually decoding field values. The data parameter should be the
// field data (not including the struct header).
// Returns the offset index sorted by field number.
func (s *Struct) scanFieldOffsets(data []byte) ([]fieldOffset, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// Pre-allocate with a reasonable estimate
	offsets := make([]fieldOffset, 0, len(s.mapping.Fields))

	var lastFieldNum int32 = -1
	offset := uint32(0)

	for offset < uint32(len(data)) {
		if uint32(len(data))-offset < 8 {
			return nil, fmt.Errorf("scanFieldOffsets: not enough bytes for field header at offset %d", offset)
		}

		h := GenericHeader(data[offset : offset+8])
		fieldNum := h.FieldNum()
		fieldType := field.Type(h.FieldType())

		// Validate field ordering
		if int32(fieldNum) <= lastFieldNum {
			return nil, fmt.Errorf("scanFieldOffsets: field %d came after field %d", fieldNum, lastFieldNum)
		}
		lastFieldNum = int32(fieldNum)

		// Calculate field size based on type
		var fieldSize uint32
		switch fieldType {
		case field.FTBool:
			fieldSize = 8
		case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
			fieldSize = 8
		case field.FTInt64, field.FTUint64, field.FTFloat64:
			fieldSize = 16
		case field.FTString, field.FTBytes:
			dataSize := h.Final40()
			fieldSize = uint32(8 + SizeWithPadding(dataSize))
		case field.FTStruct:
			structSize := h.Final40()
			fieldSize = uint32(structSize)
		case field.FTListBools:
			items := h.Final40()
			wordsNeeded := (items / 64) + 1
			fieldSize = uint32(8 + (wordsNeeded * 8))
		case field.FTListInt8, field.FTListUint8:
			items := h.Final40()
			fieldSize = uint32(8 + SizeWithPadding(items))
		case field.FTListInt16, field.FTListUint16:
			items := h.Final40()
			fieldSize = uint32(8 + SizeWithPadding(items*2))
		case field.FTListInt32, field.FTListUint32, field.FTListFloat32:
			items := h.Final40()
			fieldSize = uint32(8 + SizeWithPadding(items*4))
		case field.FTListInt64, field.FTListUint64, field.FTListFloat64:
			items := h.Final40()
			fieldSize = uint32(8 + SizeWithPadding(items*8))
		case field.FTListBytes, field.FTListStrings:
			// For list of bytes, we need to scan through each entry
			fieldSize = uint32(s.scanListBytesSize(data[offset:]))
		case field.FTListStructs:
			// For list of structs, we need to scan through each struct
			fieldSize = uint32(s.scanListStructsSize(data[offset:]))
		default:
			return nil, fmt.Errorf("scanFieldOffsets: unknown field type %v", fieldType)
		}

		offsets = append(offsets, fieldOffset{
			fieldNum: fieldNum,
			offset:   offset, // Offset relative to field data (rawData[8:])
			size:     fieldSize,
		})

		offset += fieldSize
	}

	return offsets, nil
}

// scanListBytesSize calculates the total size of a list of bytes/strings field.
func (s *Struct) scanListBytesSize(data []byte) int {
	if len(data) < 8 {
		return 0
	}
	h := GenericHeader(data[:8])
	numItems := h.Final40()
	if numItems == 0 {
		return 8
	}

	size := 8 // header
	remaining := data[8:]

	for i := uint64(0); i < numItems; i++ {
		if len(remaining) < 4 {
			return size
		}
		itemSize := int(binary.Get[uint32](remaining[:4]))
		size += 4 + itemSize
		if len(remaining) >= 4+itemSize {
			remaining = remaining[4+itemSize:]
		} else {
			break
		}
	}

	// Add padding
	paddingNeeded := PaddingNeeded(size)
	return size + paddingNeeded
}

// scanListStructsSize calculates the total size of a list of structs field.
func (s *Struct) scanListStructsSize(data []byte) int {
	if len(data) < 8 {
		return 0
	}
	h := GenericHeader(data[:8])
	numItems := h.Final40()
	if numItems == 0 {
		return 8
	}

	size := 8 // header
	remaining := data[8:]

	for i := uint64(0); i < numItems; i++ {
		if len(remaining) < 8 {
			return size
		}
		structHeader := GenericHeader(remaining[:8])
		structSize := int(structHeader.Final40())
		size += structSize
		if len(remaining) >= structSize {
			remaining = remaining[structSize:]
		} else {
			break
		}
	}

	return size
}

func (s *Struct) Unmarshal(r io.Reader) (int, error) {
	read := 0
	h := header.New()
	read, err := io.ReadFull(r, h[:])
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
	buffer := make([]byte, size-8) // -8 because we read the header already

	// Initialize lazy decode infrastructure
	s.fieldStates = make([]fieldState, len(s.mapping.Fields))
	s.modified = false
	s.decoding = true // Prevent Set* functions from applying lazy decode logic

	if len(buffer) == 0 {
		// Empty struct - no fields, but still set up rawData with just the header
		s.rawData = make([]byte, 8)
		copy(s.rawData, h)
		s.offsets = nil
		s.decoding = false
		return read, nil
	}

	n, err := io.ReadFull(r, buffer)
	read += n
	if err != nil {
		log.Println("this is the buffer size: ", len(buffer))
		return read, fmt.Errorf("problem reading Struct data: %w", err)
	}
	if int(size-8) != n {
		panic(fmt.Sprintf("read %d bytes, expected %d", n, size-8))
	}

	// Store complete raw data (header + field data) for potential fast-path marshal
	s.rawData = make([]byte, size)
	copy(s.rawData[:8], h)
	copy(s.rawData[8:], buffer)

	// Build field offset index for lazy decode support
	offsets, err := s.scanFieldOffsets(buffer)
	if err != nil {
		return read, fmt.Errorf("failed to scan field offsets: %w", err)
	}
	s.offsets = offsets

	log.Println("struct read ", read)

	// For now, still do eager decoding. Phase 3 will make this lazy.
	bufferCopy := buffer // unmarshalFields modifies the slice, so use a copy reference
	err = s.unmarshalFields(&bufferCopy)
	if err != nil {
		return read, err
	}

	// Mark all decoded fields as stateDecoded
	for _, off := range s.offsets {
		if int(off.fieldNum) < len(s.fieldStates) {
			s.fieldStates[off.fieldNum] = stateDecoded
		}
	}

	s.decoding = false // Done decoding, now Set* functions can apply lazy decode logic

	st := atomic.LoadInt64(s.structTotal)
	if read != int(st) {
		return read, fmt.Errorf("Struct was %d in length, but only found %d worth of fields", read, st)
	}

	return read, nil
}

func (s *Struct) unmarshalFields(buffer *[]byte) error {
	maxFields := uint16(len(s.mapping.Fields))
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
			XXXAddToTotal(s, len(s.excess))
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
	f.Header = (*buffer)[0:8]
	s.fields[fieldNum] = f
	XXXAddToTotal(s, 8)
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
		f.Header = (*buffer)[:8]
		s.fields[fieldNum] = f
		XXXAddToTotal(s, 8)
		*buffer = (*buffer)[8:]
	case 64:
		if len(*buffer) < 16 {
			return fmt.Errorf("can't decode a 64 bit number with < 128 bits")
		}
		f := s.fields[fieldNum]
		f.Header = (*buffer)[:8]
		v := (*buffer)[8:16]
		f.Ptr = unsafe.Pointer(&v)
		s.fields[fieldNum] = f
		XXXAddToTotal(s, 16)
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

	f.Header = (*buffer)[:8]
	b := (*buffer)[8 : 8+size] // from end of header to end of data without padding
	f.Ptr = unsafe.Pointer(&b)

	s.fields[fieldNum] = f
	log.Println("addToTotal: ", withPadding)
	XXXAddToTotal(s, withPadding)
	*buffer = (*buffer)[withPadding:]
	return nil
}

func (s *Struct) decodeListBool(buffer *[]byte, fieldNum uint16) error {
	h, ptr, err := NewBoolsFromBytes(buffer, s) // This handles our additions to s.structTotal
	if err != nil {
		return err
	}

	f := s.fields[fieldNum]
	f.Header = h
	f.Ptr = unsafe.Pointer(ptr)
	s.fields[fieldNum] = f
	return nil
}

func (s *Struct) decodeListBytes(buffer *[]byte, fieldNum uint16) error {
	f := s.fields[fieldNum]

	ptr, err := NewBytesFromBytes(buffer, s)
	if err != nil {
		return err
	}
	f.Header = ptr.header
	f.Ptr = unsafe.Pointer(ptr)
	s.fields[fieldNum] = f
	return nil
}

func (s *Struct) decodeListNumber(buffer *[]byte, fieldNum uint16) error {
	m := s.mapping.Fields[int(fieldNum)]
	f := s.fields[fieldNum]
	f.Header = (*buffer)[:8]
	var uptr unsafe.Pointer
	switch m.Type {
	case field.FTListInt8:
		ptr, err := NewNumbersFromBytes[int8](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt16:
		ptr, err := NewNumbersFromBytes[int16](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt32:
		ptr, err := NewNumbersFromBytes[int32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt64:
		ptr, err := NewNumbersFromBytes[int64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint8:
		ptr, err := NewNumbersFromBytes[uint8](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint16:
		ptr, err := NewNumbersFromBytes[uint16](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint32:
		ptr, err := NewNumbersFromBytes[uint32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint64:
		ptr, err := NewNumbersFromBytes[uint64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListFloat32:
		ptr, err := NewNumbersFromBytes[float32](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	case field.FTListFloat64:
		ptr, err := NewNumbersFromBytes[float64](buffer, s)
		if err != nil {
			return err
		}
		uptr = unsafe.Pointer(ptr)
	default:
		panic(fmt.Sprintf("Struct.decodeListNumber() called with field that is mapped to value with type: %v", m.Type))
	}
	f.Ptr = uptr
	s.fields[fieldNum] = f
	return nil
}

func (s *Struct) decodeStruct(buffer *[]byte, fieldNum uint16) error {
	// We need the mapping for the sub Struct.
	m := s.mapping.Fields[fieldNum].Mapping
	if m == nil { // This means that the contained Struct is the same mapping as the part.
		m = s.mapping
	}

	// Structs use a Reader, so let's give it a reader.
	r := readers.Get(context.Background())
	r.Reset(*buffer)
	defer readers.Put(context.Background(), r)

	sub := New(fieldNum, m)
	n, err := sub.Unmarshal(r)
	if err != nil {
		return err
	}
	SetStruct(s, fieldNum, sub)

	*buffer = (*buffer)[n:]
	return nil
}

func (s *Struct) decodeListStruct(buffer *[]byte, fieldNum uint16) error {
	// We need the mapping for the sub Struct.
	m := s.mapping.Fields[fieldNum].Mapping

	f := s.fields[fieldNum]
	log.Println("buffer size before: ", len(*buffer))
	l, err := NewStructsFromBytes(buffer, s, m)
	if err != nil {
		log.Println("buffer size after: ", len(*buffer))
		return err
	}
	f.Header = l.header
	f.Ptr = unsafe.Pointer(l)
	s.fields[fieldNum] = f
	return nil
}
