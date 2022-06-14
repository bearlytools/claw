// Package structs contains objects, functions and methods that are related to reading
// and writing Claw struct types from wire encoding.
package structs

import (
	"fmt"
	"io"
	"math"
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

// GenericHeader is the header of struct.
type GenericHeader []byte

func (g GenericHeader) First16() uint16 {
	return binary.Get[uint16](g[:2])
}

func (g GenericHeader) SetFirst16(u uint16) {
	binary.Put(g[:2], u)
}

func (g GenericHeader) Next8() uint8 {
	return g[2]
}

func (g GenericHeader) SetNext8(u uint8) {
	g[2] = u
}

func (g GenericHeader) Final40() uint64 {
	u := binary.Get[uint64](g)
	return bits.GetValue[uint64, uint64](u, dataSizeMask, 24)
}

func (g GenericHeader) SetFinal40(u uint64) {
	if u > maxDataSize {
		panic(fmt.Sprintf("can't put %d in a 40bit register, max value is 1099511627775", u))
	}
	store := binary.Get[uint64](g)
	bits.SetValue[uint64, uint64](u, store, 24, 64)
	binary.Put[uint64](g, store)
}

// structField holds a struct field entry.
type structField struct {
	header GenericHeader
	ptr    unsafe.Pointer
}

// Struct is the basic type for holding a set of values. In claw format, every variable
// must be contained in a Struct.
type Struct struct {
	header GenericHeader
	fields []structField
	excess []byte

	// mapping holds our Mapping object that allows us to understand what field number holds what value type.
	mapping mapping.Map

	parent *Struct

	// structTotal is the total size of this struct in bytes.
	structTotal *int64
}

// New creates a NewStruct that is used to create a *Struct for a specific data type.
func New(fieldNum uint16, dataMap mapping.Map, parent *Struct) *Struct {
	h := GenericHeader(make([]byte, 8))
	h.SetFirst16(fieldNum)
	h.SetNext8(uint8(field.FTStruct))

	s := &Struct{
		header:      h,
		mapping:     dataMap,
		fields:      make([]structField, len(dataMap)),
		structTotal: new(int64),
		parent:      parent,
	}
	return s
}

// NewFromReader creates a new Struct from data we read in.
func NewFromReader(r io.Reader, maps mapping.Map) (*Struct, error) {
	s := New(0, maps, nil)

	if err := s.unmarshal(r); err != nil {
		return nil, err
	}
	return s, nil
}

// IsSet determines if our Struct has a field set or not. If the fieldNum is invalid,
// this simply returns false.
func (s *Struct) IsSet(fieldNum uint16) bool {
	if int(fieldNum) > len(s.mapping) {
		return false
	}
	return s.fields[fieldNum-1].header != nil
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
	if f.header == nil {
		return false, nil
	}

	i := binary.Get[uint64](f.header)
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
	if f.header == nil {
		var n uint64
		f.header = GenericHeader(make([]byte, 8))
		n = bits.SetValue(fieldNum, n, 0, 16)
		n = bits.SetValue(uint8(field.FTBool), n, 16, 24)
		if value {
			n = bits.SetBit(n, 24, true)
		}
		binary.Put(f.header, n)
		s.fields[fieldNum-1] = f
		addToTotal(s, 8)
		return nil
	}

	n := binary.Get[uint64](f.header)
	n = bits.SetBit(n, 25, value)
	binary.Put(f.header, n)
	s.fields[fieldNum-1] = f
	return nil
}

// DeleteBool deletes a boolean and updates our storage total.
func DeleteBool(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}
	s.fields[fieldNum-1].header = nil
	return nil
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
	if f.header == nil {
		return 0, nil
	}

	if size < 64 {
		b := f.header[3:7]
		if isFloat {
			i := binary.Get[uint32](b)
			return N(math.Float32frombits(uint32(i))), nil
		}
		return N(binary.Get[uint32](b)), nil
	}
	b := *(*[]byte)(f.ptr)
	if isFloat {
		i := binary.Get[uint64](b)
		return N(math.Float64frombits(uint64(i))), nil
	}
	return N(binary.Get[uint64](b)), nil
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
	if f.header == nil {
		f.header = GenericHeader(make([]byte, 8))
		switch size < 64 {
		case true:
			addToTotal(s, 8)
		case false:
			b := make([]byte, 8)
			f.ptr = unsafe.Pointer(&b)
			addToTotal(s, 16)
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
			binary.Put(f.header[:8], ints[0])
		case false:
			i := math.Float64bits(float64(value))
			ints[1] = i
			binary.Put(f.header[:8], ints[0])
			d := (*[]byte)(f.ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	} else {
		// Now encode the Number.
		switch size < 64 {
		case true:
			ints[0] = bits.SetValue(uint32(value), ints[0], 24, 64)
			binary.Put(f.header[:8], ints[0])
		case false:
			ints[1] = uint64(value)
			binary.Put(f.header[:8], ints[0])
			d := (*[]byte)(f.ptr)
			binary.Put(*d, ints[1])
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
		addToTotal(s, -8)
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		addToTotal(s, -16)
	default:
		panic("wtf")
	}
	f := s.fields[fieldNum-1]
	f.header = nil
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetBytes returns a field of bytes (also our string as well in []byte form). If the value was not
// set, this is returned as nil. If it was set, but empty, this will be []byte{}.
func GetBytes(s *Struct, fieldNum uint16) ([]byte, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil { // The zero value
		return nil, nil
	}

	if f.ptr == nil { // Set, but value is empty
		return []byte{}, nil
	}

	x := (*[]byte)(f.ptr)
	return *x, nil
}

// SetBytes sets a field of bytes (also our string as well in []byte form).
func SetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}

	if len(value) > maxDataSize {
		return fmt.Errorf("cannot set a String or Byte field to size > 1099511627775")
	}

	f := s.fields[fieldNum-1]
	if value == nil {
		if f.header == nil { // It is already unset
			return nil
		}
		remove := 8
		if f.ptr != nil {
			x := (*[]byte)(f.ptr)
			dataSize := len(*x)
			// Data stored may or may not have padding, so if not aligned this will
			// add the padding.
			remove += SizeWithPadding(dataSize)
		}
		addToTotal(s, -remove)
		f.header = nil
		f.ptr = nil
		s.fields[fieldNum-1] = f
		return nil
	}

	ftype := field.FTBytes
	if isString {
		ftype = field.FTString
	}

	remove := 0
	// If the field isn't allocated, allocate space.
	if f.header == nil {
		f.header = GenericHeader(make([]byte, 8))
	} else { // We need to remove our existing entry size total before applying our new data
		remove += 8 + SizeWithPadding(int(f.header.Final40()))
		addToTotal(s, -remove)
	}
	f.header.SetFirst16(fieldNum)
	f.header.SetNext8(uint8(ftype))
	f.header.SetFinal40(uint64(len(value)))

	f.ptr = unsafe.Pointer(&value)
	// We don't store any padding at this point because we don't want to do another allocation.
	// But we do record the size it would be with padding.
	addToTotal(s, int64(8+SizeWithPadding(len(value))))
	s.fields[fieldNum-1] = f
	return nil
}

// DeleteBytes deletes the bytes field and updates our storage total.
func DeleteBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}
	remove := 8
	f.header = nil
	if f.ptr == nil {
		addToTotal(s, -remove)
		return nil
	}
	x := (*[]byte)(f.ptr)
	remove += SizeWithPadding(len(*x))
	addToTotal(s, -remove)
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

func addToTotal[N int64 | int | uint | uint64](s *Struct, value N) {
	atomic.AddInt64(s.structTotal, int64(value))
	var ptr *Struct
	for {
		ptr = s.parent
		if ptr == nil {
			return
		}
		atomic.AddInt64(ptr.structTotal, int64(value))
	}
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
