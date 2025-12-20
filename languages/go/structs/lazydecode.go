package structs

import (
	"unsafe"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/gostdlib/base/context"
)

func init() {
	mapping.RegisterLazyDecoders = registerLazyDecoders
}

func registerLazyDecoders(m *mapping.Map) {
	m.LazyDecoders = make([]mapping.LazyDecodeFunc, len(m.Fields))
	for i, f := range m.Fields {
		m.LazyDecoders[i] = lazyDecoderForType(f.Type)
	}
}

func lazyDecoderForType(t field.Type) mapping.LazyDecodeFunc {
	switch t {
	case field.FTBool:
		return lazyDecodeScalar8
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		return lazyDecodeScalar8
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		return lazyDecodeScalar64
	case field.FTString, field.FTBytes:
		return lazyDecodeBytes
	case field.FTStruct:
		return lazyDecodeStruct
	case field.FTListBools:
		return lazyDecodeListBools
	case field.FTListInt8:
		return lazyDecodeListInt8
	case field.FTListInt16:
		return lazyDecodeListInt16
	case field.FTListInt32:
		return lazyDecodeListInt32
	case field.FTListInt64:
		return lazyDecodeListInt64
	case field.FTListUint8:
		return lazyDecodeListUint8
	case field.FTListUint16:
		return lazyDecodeListUint16
	case field.FTListUint32:
		return lazyDecodeListUint32
	case field.FTListUint64:
		return lazyDecodeListUint64
	case field.FTListFloat32:
		return lazyDecodeListFloat32
	case field.FTListFloat64:
		return lazyDecodeListFloat64
	case field.FTListBytes, field.FTListStrings:
		return lazyDecodeListBytes
	case field.FTListStructs:
		return lazyDecodeListStructs
	default:
		return lazyDecodeNoop
	}
}

// lazyDecodeScalar8 decodes scalar types that fit entirely in the 8-byte header.
func lazyDecodeScalar8(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
}

// lazyDecodeScalar64 decodes 64-bit scalar types (header + 8 bytes of data).
func lazyDecodeScalar64(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	if len(data) >= 16 {
		v := data[8:16]
		f.Ptr = unsafe.Pointer(&v)
	}
}

// lazyDecodeBytes decodes string and bytes fields.
func lazyDecodeBytes(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	h := GenericHeader(data[:8])
	size := h.Final40()
	if size > 0 && len(data) >= int(8+size) {
		b := data[8 : 8+size]
		f.Ptr = unsafe.Pointer(&b)
	}
}

// lazyDecodeStruct decodes a nested struct field.
func lazyDecodeStruct(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	m := desc.Mapping
	if m == nil {
		m = s.mapping // Self-referential
	}

	sub := New(fieldNum, m)
	// Create a reader from the data and unmarshal
	r := readers.Get(context.Background())
	r.Reset(data)
	defer readers.Put(context.Background(), r)

	_, err := sub.Unmarshal(r)
	if err != nil {
		return
	}

	sub.parent = s
	f := &s.fields[fieldNum]
	f.Header = sub.header
	f.Ptr = unsafe.Pointer(sub)
}

// lazyDecodeListBools decodes a list of bools.
func lazyDecodeListBools(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	dataCopy := data
	h, ptr, err := NewBoolsFromBytes(&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f := &s.fields[fieldNum]
	f.Header = h
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListInt8 decodes a list of int8.
func lazyDecodeListInt8(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[int8](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListInt16 decodes a list of int16.
func lazyDecodeListInt16(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[int16](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListInt32 decodes a list of int32.
func lazyDecodeListInt32(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[int32](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListInt64 decodes a list of int64.
func lazyDecodeListInt64(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[int64](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListUint8 decodes a list of uint8.
func lazyDecodeListUint8(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[uint8](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListUint16 decodes a list of uint16.
func lazyDecodeListUint16(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[uint16](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListUint32 decodes a list of uint32.
func lazyDecodeListUint32(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[uint32](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListUint64 decodes a list of uint64.
func lazyDecodeListUint64(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[uint64](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListFloat32 decodes a list of float32.
func lazyDecodeListFloat32(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[float32](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListFloat64 decodes a list of float64.
func lazyDecodeListFloat64(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	f := &s.fields[fieldNum]
	f.Header = data[:8]
	dataCopy := data
	ptr, err := NewNumbersFromBytes[float64](&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListBytes decodes a list of bytes or strings.
func lazyDecodeListBytes(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	dataCopy := data
	ptr, err := NewBytesFromBytes(&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f := &s.fields[fieldNum]
	f.Header = ptr.header
	f.Ptr = unsafe.Pointer(ptr)
}

// lazyDecodeListStructs decodes a list of structs.
func lazyDecodeListStructs(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
	s := (*Struct)(structPtr)
	m := desc.Mapping
	dataCopy := data
	l, err := NewStructsFromBytes(&dataCopy, nil, m)
	if err != nil {
		return
	}
	l.s = s
	f := &s.fields[fieldNum]
	f.Header = l.header
	f.Ptr = unsafe.Pointer(l)
}

// lazyDecodeNoop does nothing for unknown field types.
func lazyDecodeNoop(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *mapping.FieldDescr) {
}
