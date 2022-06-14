// Package structs contains objects, functions and methods that are related to reading
// and writing Claw struct types from wire encoding.
package structs

import (
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
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

	// total is the total size of the top level Struct. It is passed from the top level all
	// the way down.
	total *int64
}

// NewStruct creates a *Struct that represents a specific defined *Struct that our user defined.
// It is a kind of "factory" (I hate to use that term because we are in Go) that has all our
// internals already presized and ready before we start, instead of computing it each time.
// This type is created with New().
type NewStruct func(fieldNum uint16) *Struct

// New creates a NewStruct that is used to create a *Struct for a specific data type.
func New(dataMap mapping.Map) NewStruct {
	s := Struct{
		header:  Header{FieldType: field.FTStruct},
		mapping: dataMap,
		// TODO(jdoak): replace with a pull from a generated pool.
		fields:           make([][]byte, len(dataMap)),
		fieldNumToStruct: map[uint16]int{},
		fieldNumToList:   map[uint16]int{},
		total:            new(int64),
	}

	structs := 0
	lists := 0
	for fieldNum, fieldDesc := range dataMap {
		switch fieldDesc.Type {
		case field.FTBool:
		case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		case field.FTInt64, field.FTUint64, field.FTFloat64:
		case field.FTBytes, field.FTString:
		case field.FTStruct:
			s.fieldNumToStruct[uint16(fieldNum)] = structs
			structs++
		case field.FTListBool, field.FTListBytes, field.FTListStruct, field.FTList8, field.FTList16, field.FTList32, field.FTList64:
			s.fieldNumToList[uint16(fieldNum)] = lists
			lists++
		default:
			panic(fmt.Sprintf("bug: the dataMap passed had a type %v that we don't support", fieldDesc.Type))
		}
	}
	s.structs = make([]*Struct, structs)
	s.lists = make([]unsafe.Pointer, lists)

	return func(fieldNum uint16) *Struct {
		n := copyStruct(s)
		total := *s.total
		n.total = &total
		// TODO(jdoak): These can come out of a pool.
		n.fields = make([][]byte, len(n.fields))
		n.structs = make([]*Struct, len(n.structs))
		n.lists = make([]unsafe.Pointer, len(n.lists))
		return &n
	}
}

// Args are arguments to NewFromReader().
type Args struct {
	Data                     io.Reader
	Map                      mapping.Map
	Fields                   [][]byte
	StructLookup, ListLookup map[uint16]int
}

// NewFromReader creates a new Struct from data we read in.
func NewFromReader(args Args, bufferPool *sync.Pool) (*Struct, error) {
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

	s := &Struct{
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
		return nil, nil
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
		return s.structs[s.fieldNumToStruct[fieldNum]] != nil
	case field.FTListStruct, field.FTListBool, field.FTListBytes, field.FTList8, field.FTList16, field.FTList32, field.FTList64:
		return s.lists[s.fieldNumToList[fieldNum]] != nil
	}

	// Its a non special field type, so simply simply return if it is not nil (aka set).
	return s.fields[fieldNum] != nil
}

var boolMask = bits.Mask[uint64](24, 25)

// GetBool gets a bool value from field at fieldNum. This return an error if the field
// is not a bool or fieldNum is not a valid field number. If the field is not set, it
// returns false with no error.
func GetBool(s *Struct, fieldNum uint16) (bool, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return false, err
	}

	f := s.fields[fieldNum-1]
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

// SetBool sets a boolean value in field "fieldNum" to value "value".
func SetBool(s *Struct, fieldNum uint16, value bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f == nil {
		var n uint64
		f = make([]byte, 8)
		n = bits.SetValue(fieldNum, n, 0, 16)
		n = bits.SetValue(uint8(field.FTBool), n, 16, 24)
		if value {
			n = bits.SetBit(n, 24, true)
		}
		binary.Put(f, n)
		s.fields[fieldNum-1] = f
		atomic.AddInt64(s.total, 8)
		return nil
	}

	n := binary.Get[uint64](f)
	n = bits.SetBit(n, 25, value)
	binary.Put(f, n)
	s.fields[fieldNum-1] = f
	return nil
}

// DeleteBool deletes a boolean and updates our storage total.
func DeleteBool(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}
	s.fields[fieldNum-1] = nil
	return nil
}

var numTypes = []field.Type{
	field.FTInt8,
	field.FTInt16,
	field.FTInt32,
	field.FTInt64,
	field.FTUint8,
	field.FTUint16,
	field.FTUint32,
	field.FTUint64,
	field.FTFloat32,
	field.FTFloat64,
}

// GetNumber gets a number value at fieldNum.
func GetNumber[N Numbers](s *Struct, fieldNum uint16) (N, error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return 0, err
	}
	desc := s.mapping[fieldNum-1]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return 0, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum-1]
	if f == nil {
		return 0, nil
	}

	if size < 64 {
		f = f[3:7]
		if isFloat {
			i := binary.Get[uint32](f)
			return N(math.Float32frombits(uint32(i))), nil
		}
		return N(binary.Get[uint32](f)), nil
	}
	f = f[8:16]
	if isFloat {
		i := binary.Get[uint64](f)
		return N(math.Float64frombits(uint64(i))), nil
	}
	return N(binary.Get[uint64](f)), nil
}

// SetNumber sets a number value in field "fieldNum" to value "value".
func SetNumber[N Numbers](s *Struct, fieldNum uint16, value N) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping[fieldNum-1]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum-1]
	// If the field isn't allocated, allocate space.
	if f == nil {
		switch size < 64 {
		case true:
			f = make([]byte, 8)
			atomic.AddInt64(s.total, 8)
		case false:
			f = make([]byte, 16)
			atomic.AddInt64(s.total, 16)
		default:
			panic("wtf")
		}
	}

	// Its will store up to 2 uint64s that will be written. 1 is written if we can fit our value
	// in the header, 2 if we can't.
	ints := [2]uint64{}
	// Write our header information.
	ints[0] = bits.SetValue(fieldNum, ints[0], 0, 16)
	ints[0] = bits.SetValue(uint8(desc.Type), ints[0], 16, 24)

	// If the number is a float, convert it to the uint representation.
	if isFloat {
		switch size < 64 {
		case true:
			i := math.Float32bits(float32(value))
			ints[0] = bits.SetValue(i, ints[0], 24, 64)
			binary.Put(f[:8], ints[0])
		case false:
			i := math.Float64bits(float64(value))
			ints[1] = i
			binary.Put(f[:8], ints[0])
			binary.Put(f[8:], ints[1])
		default:
			panic("wtf")
		}
	} else {
		// Now encode the Number.
		switch size < 64 {
		case true:
			ints[0] = bits.SetValue(uint32(value), ints[0], 24, 64)
			binary.Put(f[:8], ints[0])
		case false:
			ints[1] = uint64(value)
			binary.Put(f[:8], ints[0])
			binary.Put(f[8:], ints[1])
		default:
			panic("wtf")
		}
	}
	s.fields[fieldNum-1] = f
	return nil
}

// DeleteNumber deletes the number and updates our storage total.
func DeleteNumber(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping[fieldNum-1]

	switch desc.Type {
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		atomic.AddInt64(s.total, -8) // Requires single 64 bit value (8 bytes)
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		atomic.AddInt64(s.total, -16) // Requires single two 64 bit value (16 bytes)
	default:
		panic("wtf")
	}
	s.fields[fieldNum-1] = nil
	return nil
}

/*
func SetBytes(s *Struct, fieldNum uint16, value []byte) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}
	desc := s.mapping[fieldNum-1]

	if len(value) > maxDataSize {
		return fmt.Errorf("cannot set a String or Byte field to size > 1099511627775")
	}

	if value == nil {
		atomic.AddInt64(s.total, -int64(len(s.fields[fieldNum-1])))
		s.fields[fieldNum-1] = nil
	}

	f := s.fields[fieldNum-1]
	// If the field isn't allocated, allocate space.
	if f == nil {
		f = make([][]byte, 2)
		f[0] = make([]byte, 8)

		var u uint64
		u = bits.SetValue[uint16, uint64](fieldNum, u, 0, 16)
		u = bits.SetValue[uint8, uint64](uint8(desc.Type), u, 16, 24)
		u = bits.SetValue[uint64, uint64](uint64(len(value)), u, 24, 64)
		binary.Put(f[0], u)
	} else {
		u = binary.Get[uint64](f[0])
		u = bits.SetValue[uint64, uint64](uint64(len(value)), u, 24, 64)
		binary.Put(f[0], u)
	}
	f[1] = value
	s.bytesField[fieldNum-1] = f
	return nil
}
*/

// copyStruct is a helper that makes a copy of a Struct.
func copyStruct(s Struct) Struct {
	return s
}

// validateFieldNum will validate that the fieldNum is > 0, that the type is described in the mapping.Map,
// and if len(ftypes) != 0, that the ftype and mapping.Map[fieldNum].Type are the same.
func validateFieldNum(fieldNum uint16, maps mapping.Map, ftypes ...field.Type) error {
	if fieldNum == 0 {
		return fmt.Errorf("fieldNum cannot be 0")
	}
	if int(fieldNum) > len(maps) {
		return fmt.Errorf("fieldNum is > the number of possible fields")
	}
	if len(ftypes) == 0 {
		return nil
	}

	desc := maps[fieldNum-1]
	found := false
	for _, ftype := range ftypes {
		if desc.Type == ftype {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("fieldNum(%d) was %v, which was not valid", fieldNum, desc.Type)
	}
	return nil
}

func numberToDescCheck[N Numbers](desc *mapping.FieldDesc) (size uint8, isFloat bool, err error) {
	var t N
	switch any(t).(type) {
	case uint8:
		if desc.Type != field.FTUint8 {
			return 0, false, fmt.Errorf("fieldNum is not a uint8 type, was %v", desc.Type)
		}
		size = 8
	case uint16:
		if desc.Type != field.FTUint16 {
			return 0, false, fmt.Errorf("fieldNum is not a uint16 type, was %v", desc.Type)
		}
		size = 16
	case uint32:
		if desc.Type != field.FTUint32 {
			return 0, false, fmt.Errorf("fieldNum is not a uint32 type, was %v", desc.Type)
		}
		size = 32
	case uint64:
		if desc.Type != field.FTUint64 {
			return 0, false, fmt.Errorf("fieldNum is not a uint64 type, was %v", desc.Type)
		}
		size = 64
	case int8:
		if desc.Type != field.FTInt8 {
			return 0, false, fmt.Errorf("fieldNum is not a int8 type, was %v", desc.Type)
		}
		size = 8
	case int16:
		if desc.Type != field.FTInt16 {
			return 0, false, fmt.Errorf("fieldNum is not a int16 type, was %v", desc.Type)
		}
		size = 16
	case int32:
		if desc.Type != field.FTInt32 {
			return 0, false, fmt.Errorf("fieldNum is not a int32 type, was %v", desc.Type)
		}
		size = 32
	case int64:
		if desc.Type != field.FTInt64 {
			return 0, false, fmt.Errorf("fieldNum is not a int64 type, was %v", desc.Type)
		}
		size = 64
	case float32:
		if desc.Type != field.FTFloat32 {
			return 0, false, fmt.Errorf("fieldNum is not a float32 type, was %v", desc.Type)
		}
		size = 32
		isFloat = true
	case float64:
		if desc.Type != field.FTFloat64 {
			return 0, false, fmt.Errorf("fieldNum is not a float64 type, was %v", desc.Type)
		}
		size = 64
		isFloat = true
	default:
		return 0, false, fmt.Errorf("passed a number value of %T that we do not support", t)
	}
	return size, isFloat, nil
}
