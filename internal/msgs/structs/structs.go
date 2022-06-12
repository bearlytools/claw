// Package structs contains objects, functions and methods that are related to reading
// and writing Claw struct types from wire encoding.
package structs

import (
	"fmt"
	"io"
	"sync"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/internal/mapping"
)

const (
	// maxDataSize is the max number that can fit into the dataSize field, which is 40 bits.
	maxDataSize = 1099511627775
	// fieldtype is the struct field type, circular dependency prevents the
	// constant import from claw.go
	fieldType uint8 = 14
)

// Masks to use to pull information from a bitpacked uint64.
var (
	fieldNumMask  = bits.Mask[uint64](0, 16)
	fieldTypeMask = bits.Mask[uint64](16, 24)
	dataSizeMask  = bits.Mask[uint64](24, 64)
)

var headerPool = sync.Pool{
	New: func() any {
		v := make([]byte, 8)
		return &v
	},
}

// Header is the header of struct.
type Header struct {
	FieldNum  uint16
	FieldType field.Type // This is always 14, anything else and something is wrong
	DataSize  uint64     // Max value is 40 bits or 1099511627775
}

// Read unpacks a bitpacked uint64 stored in a slice of bytes into our
// Header information.
func (h *Header) Read(b []byte) error {
	if len(b) != 8 { // Headers are uint64, so 8 bytes.
		return fmt.Errorf("struct.Header.Read() must recieve a []byte of len 8")
	}

	s := binary.Get[uint64](b)

	h.FieldNum = bits.GetValue[uint64, uint16](s, fieldNumMask, 0)
	h.FieldType = field.Type(bits.GetValue[uint64, uint8](s, fieldTypeMask, 16))
	h.DataSize = bits.GetValue[uint64, uint64](s, dataSizeMask, 24)
	return h.validate()
}

// Write writes the header onto the io.Writer passed.
func (h *Header) Write(w io.Writer) (int, error) {
	b := headerPool.Get().(*[]byte)
	defer headerPool.Put(b)

	h.Bytes(*b)
	return w.Write(*b)
}

// Bytes converts the Header to a slice of 8 bytes holding the Header information.
// It should be noted that FieldType, regardless of what it is set to, will always
// be encoded as 14. If the Header is corrupted or the size of "b" is not 8,
// this is going to panic. While reading in a Header that is bad is an error, writing
// a bad Header is a critical error.
func (h Header) Bytes(b []byte) {
	if len(b) != 8 {
		panic("Header.ToBytes() requires a slice of exactly 8 bytes")
	}
	if h.DataSize > maxDataSize {
		panic("dataSize cannot be greater than 2^40 -1")
	}

	var s uint64
	s = bits.SetValue(h.FieldNum, s, 0, 16)
	s = bits.SetValue(fieldType, s, 17, 24)
	s = bits.SetValue(h.DataSize, s, 25, 64)
	binary.Put(b, s)
}

func (h Header) validate() error {
	if h.DataSize > maxDataSize {
		return fmt.Errorf("DataSize is a 40 bit number, this was %d, max is %d", h.DataSize, maxDataSize)
	}
	if h.FieldType != field.FTStruct {
		return fmt.Errorf("a struct.Header.FieldType must be %d, was %d", field.FTStruct, h.FieldType)
	}
	if h.FieldNum == 0 {
		return fmt.Errorf("a struct.Header.FieldNum must be non-zero")
	}
	return nil
}

// Struct is the basic type for holding a set of values. In claw format, every variable
// must be contained in a Struct.
type Struct struct {
	header Header
	fields [][]byte

	fieldNumToStruct map[uint16]int
	fieldNumToList   map[uint16]int
	structs          []*Struct
	lists            []unsafe.Pointer

	// mapping holds our Mapping object that allows us to understand what field number holds what value type.
	mapping mapping.Map

	// notZero is an indicator if a Struct{} is the zero value. A New() call sets this.
	notZero bool

	// total is the total size of the top level Struct. It is passed from the top level all
	// the way down.
	total *int64
}

type Args struct {
	Data                     io.Reader
	Map                      mapping.Map
	Fields                   [][]byte
	StructLookup, ListLookup map[uint16]int
}

// New creates a new Struct.
func New(args Args, bufferPool *sync.Pool) (Struct, error) {
	for i := 0; i < len(args.Fields); i++ {
		args.Fields[i] = nil
	}

	numStructs := 0
	numList := 0
	for _, desc := range args.Map {
		switch desc.Type {
		case field.FTStruct:
			numStructs++
		case field.FTListStruct, field.FTListBytes, field.FTListBool, field.FTList8, field.FTList16, field.FTList32, field.FTList64:
			numList++
		}
	}

	s := Struct{
		notZero:          true,
		mapping:          args.Map,
		fields:           args.Fields,
		structs:          make([]*Struct, numStructs),
		lists:            make([]unsafe.Pointer, numList),
		fieldNumToStruct: args.StructLookup,
		fieldNumToList:   args.ListLookup,
	}

	buff := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buff)

	if err := s.unmarshal(args.Data, buff); err != nil {
		return Struct{}, nil
	}
	return s, nil
}

// IsSet determines if our Struct has a field set or not. If the fieldNum is invalid,
// this simply returns false.
func (s *Struct) IsSet(fieldNum uint16) bool {
	if int(fieldNum) > len(s.mapping) {
		return false
	}

	// Check if we are a special type holding a Struct of list of Structs. If so,
	// Check our internal mappings instead of .fields.
	desc := s.mapping[fieldNum-1]
	switch desc.Type {
	case field.FTStruct:
		return s.structs[s.fieldNumToStruct[fieldNum]].notZero
	case field.FTListStruct, field.FTListBool, field.FTListBytes, field.FTList8, field.FTList16, field.FTList32, field.FTList64:
		return s.lists[s.fieldNumToList[fieldNum]] != nil
	}

	// Its a non special field type, so simply simply return if it is not nil (aka set).
	return s.fields[fieldNum] != nil
}

var boolMask = bits.Mask[uint64](24, 25)

// Bool gets a bool value from field at fieldNum. This return an error if the field
// is not a bool or fieldNum is not a valid field number. If the field is not set, it
// returns false with no error.
func (s Struct) Bool(fieldNum uint16) (bool, error) {
	if int(fieldNum) > len(s.mapping) {
		return false, fmt.Errorf("fieldNum is > the number of possible fields")
	}
	desc := s.mapping[fieldNum-1]
	if desc.Type != field.FTBool {
		return false, fmt.Errorf("fieldNum is not a Bool type, was %v", desc.Type)
	}

	f := s.fields[fieldNum]
	// Return the zero value of a non-set field.
	if f == nil {
		return false, nil
	}

	i := binary.Get[uint64](f)
	if bits.GetValue[uint64, uint8](i, boolMask, 24) == 1 {
		return true, nil
	}
	return false, nil
}
