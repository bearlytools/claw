// Package structs contains objects, functions and methods that are related to reading
// and writing Claw struct types from wire encoding.  THIS FILE IS PUBLIC ONLY OUT OF
// NECCESSITY AND ANY USE IS NOT PROTECTED between any versions. Seriously, your code
// will break if you use this.
package structs

import (
	"fmt"
	"io"
	"log"
	"math"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/bearlytools/claw/languages/go/structs/header"
)

const (
	// maxDataSize is the max number that can fit into the dataSize field, which is 40 bits.
	maxDataSize = 1099511627775
)

// GenericHeader is the header of struct.
type GenericHeader = header.Generic

func NewGenericHeader() GenericHeader {
	return header.New()
}

// StructField holds a struct field entry.
type StructField struct {
	Header header.Generic
	Ptr    unsafe.Pointer
}

// Struct is the basic type for holding a set of values. In claw format, every variable
// must be contained in a Struct.
type Struct struct {
	inList bool

	header GenericHeader
	fields []StructField
	excess []byte

	// mapping holds our Mapping object that allows us to understand what field number holds what value type.
	mapping *mapping.Map

	parent *Struct

	// structTotal is the total size of this struct in bytes, including header.
	structTotal *int64

	// zeroTypeCompression indicates if we want to compress the encoding by ignoring
	// scalar zero values.
	zeroTypeCompression bool
}

// New creates a NewStruct that is used to create a *Struct for a specific data type.
func New(fieldNum uint16, dataMap *mapping.Map) *Struct {
	if dataMap == nil {
		panic("dataMap must not be nil")
	}
	h := GenericHeader(make([]byte, 8))
	h.SetFieldNum(fieldNum)
	h.SetFieldType(field.FTStruct)

	s := &Struct{
		header:              h,
		mapping:             dataMap,
		fields:              make([]StructField, len(dataMap.Fields)),
		structTotal:         new(int64),
		zeroTypeCompression: true,
	}
	XXXAddToTotal(s, 8) // the header
	return s
}

// NewFromReader creates a new Struct from data we read in.
func NewFromReader(r io.Reader, maps *mapping.Map) (*Struct, error) {
	s := New(0, maps)

	if _, err := s.unmarshal(r); err != nil {
		return nil, err
	}
	return s, nil
}

// XXXSetNoZeroTypeCompression sets the Struct to output scalar value headers even if the
// value is set to the zero value of the type. This makes the size larger but allows
// detection if the field was set to 0 versus being a zero value.
// As with all XXXFunName, this is meant to be used internally. Using this otherwise
// can have bad effects and there is no compatibility promise around it.
func (s *Struct) XXXSetNoZeroTypeCompression() {
	s.zeroTypeCompression = false
}

// NewFrom creates a new Struct that represents the same Struct type.
func (s *Struct) NewFrom() *Struct {
	h := GenericHeader(make([]byte, 8))
	h.SetFieldType(field.FTStruct)

	n := &Struct{
		header:              h,
		mapping:             s.mapping,
		fields:              make([]StructField, len(s.mapping.Fields)),
		structTotal:         new(int64),
		zeroTypeCompression: s.zeroTypeCompression,
	}
	XXXAddToTotal(n, 8) // the header
	return n
}

func (s *Struct) Map() *mapping.Map {
	return s.mapping
}

// Fields returns the list of StructFields.
func (s *Struct) Fields() []StructField {
	return s.fields
}

// IsSet determines if our Struct has a field set or not. If the fieldNum is invalid,
// this simply returns false. If NoZeroTypeCompression is NOT set, then we will return
// true for all scaler values, string and bytes.
func (s *Struct) IsSet(fieldNum uint16) bool {
	if int(fieldNum) > len(s.mapping.Fields) {
		return false
	}

	// Not type compression means that we always have a header for a value, even the zero value.
	if !s.zeroTypeCompression {
		return s.fields[fieldNum].Header != nil
	}

	// Well, then if the Header isn't nil, we know it is set.
	if s.fields[fieldNum].Header != nil {
		return true
	}

	// The Header is nil, so only some types can still report if they are not set.
	t := s.mapping.Fields[int(fieldNum)].Type
	if t == field.FTStruct {
		return false
	}
	for _, lt := range field.ListTypes {
		if t == lt {
			return false
		}
	}

	return true
}

var boolMask = bits.Mask[uint64](24, 25)

// GetBool gets a bool value from field at fieldNum. This return an error if the field
// is not a bool or fieldNum is not a valid field number. If the field is not set, it
// returns false with no error.
func GetBool(s *Struct, fieldNum uint16) (bool, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return false, err
	}

	f := s.fields[fieldNum]
	// Return the zero value of a non-set field.
	if f.Header == nil {
		return false, nil
	}

	i := binary.Get[uint64](f.Header)
	if bits.GetValue[uint64, uint8](i, boolMask, 24) == 1 {
		return true, nil
	}
	return false, nil
}

func MustGetBool(s *Struct, fieldNum uint16) bool {
	b, err := GetBool(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

// SetBool sets a boolean value in field "fieldNum" to value "value".
func SetBool(s *Struct, fieldNum uint16, value bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		f.Header = NewGenericHeader()
		f.Header.SetFieldNum(fieldNum)
		f.Header.SetFieldType(field.FTBool)

		log.Println("parent: ", s.parent)
		XXXAddToTotal(s, 8)
	}
	n := conversions.BytesToNum[uint64](f.Header)
	*n = bits.SetBit(*n, 24, value)
	s.fields[fieldNum] = f
	return nil
}

func MustSetBool(s *Struct, fieldNum uint16, value bool) {
	err := SetBool(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteBool deletes a boolean and updates our storage total.
func DeleteBool(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}
	s.fields[fieldNum].Header = nil
	return nil
}

// GetNumber gets a number value at fieldNum.
func GetNumber[N Number](s *Struct, fieldNum uint16) (N, error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return 0, err
	}
	desc := s.mapping.Fields[fieldNum]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return 0, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return 0, nil
	}

	if size < 64 {
		b := f.Header[3:8]
		if isFloat {
			i := binary.Get[uint32](b)
			return N(math.Float32frombits(uint32(i))), nil
		}
		return N(binary.Get[uint32](b)), nil
	}
	b := *(*[]byte)(f.Ptr)
	if isFloat {
		i := binary.Get[uint64](b)
		return N(math.Float64frombits(uint64(i))), nil
	}
	return N(binary.Get[uint64](b)), nil
}

func MustGetNumber[N Number](s *Struct, fieldNum uint16) N {
	n, err := GetNumber[N](s, fieldNum)
	if err != nil {
		panic(err)
	}
	return n
}

// SetNumber sets a number value in field "fieldNum" to value "value".
func SetNumber[N Number](s *Struct, fieldNum uint16, value N) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping.Fields[fieldNum]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum]
	// If the field isn't allocated, allocate space.
	if f.Header == nil {
		f.Header = NewGenericHeader()
		switch size < 64 {
		case true:
			XXXAddToTotal(s, 8)
		case false:
			b := make([]byte, 8)
			f.Ptr = unsafe.Pointer(&b)
			XXXAddToTotal(s, 16)
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
			binary.Put(f.Header[:8], ints[0])
		case false:
			i := math.Float64bits(float64(value))
			ints[1] = i
			binary.Put(f.Header[:8], ints[0])
			d := (*[]byte)(f.Ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	} else {
		// Now encode the Number.
		switch size < 64 {
		case true:
			ints[0] = bits.SetValue(uint32(value), ints[0], 24, 64)
			binary.Put(f.Header[:8], ints[0])
		case false:
			ints[1] = uint64(value)
			binary.Put(f.Header[:8], ints[0])
			d := (*[]byte)(f.Ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	}
	s.fields[fieldNum] = f
	return nil
}

func MustSetNumber[N Number](s *Struct, fieldNum uint16, value N) {
	err := SetNumber[N](s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteNumber deletes the number and updates our storage total.
func DeleteNumber(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping.Fields[fieldNum]

	switch desc.Type {
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		XXXAddToTotal(s, -8)
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		XXXAddToTotal(s, -16)
	default:
		panic("wtf")
	}
	f := s.fields[fieldNum]
	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// GetBytes returns a field of bytes (also our string as well in []byte form). If the value was not
// set, this is returned as nil. If it was set, but empty, this will be []byte{}. It is UNSAFE to modify
// this.
func GetBytes(s *Struct, fieldNum uint16) (*[]byte, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum]
	if f.Header == nil { // The zero value
		return nil, nil
	}

	if f.Ptr == nil { // Set, but value is empty
		return nil, nil
	}

	x := (*[]byte)(f.Ptr)
	return x, nil
}

func MustGetBytes(s *Struct, fieldNum uint16) *[]byte {
	b, err := GetBytes(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

// SetBytes sets a field of bytes (also our string as well in []byte form).
func SetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}
	if len(value) == 0 {
		return fmt.Errorf("cannot encode an empty Bytes value")
	}

	if len(value) > maxDataSize {
		return fmt.Errorf("cannot set a String or Byte field to size > 1099511627775")
	}

	f := s.fields[fieldNum]

	ftype := field.FTBytes
	if isString {
		ftype = field.FTString
	}

	remove := 0
	// If the field isn't allocated, allocate space.
	if f.Header == nil {
		f.Header = NewGenericHeader()
	} else { // We need to remove our existing entry size total before applying our new data
		remove += 8 + SizeWithPadding(int(f.Header.Final40()))
		XXXAddToTotal(s, -remove)
	}
	f.Header.SetFieldNum(fieldNum)
	f.Header.SetFieldType(ftype)
	f.Header.SetFinal40(uint64(len(value)))

	f.Ptr = unsafe.Pointer(&value)
	// We don't store any padding at this point because we don't want to do another allocation.
	// But we do record the size it would be with padding.
	XXXAddToTotal(s, int64(8+SizeWithPadding(len(value))))
	s.fields[fieldNum] = f
	return nil
}

func MustSetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) {
	err := SetBytes(s, fieldNum, value, isString)
	if err != nil {
		panic(err)
	}
}

// DeleteBytes deletes a bytes field and updates our storage total.
func DeleteBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil
	}

	f.Header = nil
	x := (*[]byte)(f.Ptr)
	XXXAddToTotal(s, -len(*x))
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// GetStruct returns a Struct field . If the value was not set, this is returned as nil. If it was set,
// but empty, this will be *Struct with no data.
func GetStruct(s *Struct, fieldNum uint16) (*Struct, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum]
	if f.Header == nil { // The zero value
		return nil, nil
	}

	x := (*Struct)(f.Ptr)
	return x, nil
}

func MustGetStruct(s *Struct, fieldNum uint16) *Struct {
	s, err := GetStruct(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return s
}

// SetStruct sets a Struct field.
func SetStruct(s *Struct, fieldNum uint16, value *Struct) error {
	if s == nil {
		return fmt.Errorf("value cannot be added to a nil Struct")
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil, to delete a Struct use DeleteStruct()")
	}
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return err
	}

	if atomic.LoadInt64(value.structTotal) > maxDataSize {
		return fmt.Errorf("cannot set a Struct field to size > 1099511627775")
	}

	f := s.fields[fieldNum]

	value.parent = s
	value.header.SetFieldNum(fieldNum)

	var remove int64
	// We need to remove our existing entry size total before applying our new data
	if f.Header != nil {
		x := (*Struct)(f.Ptr)
		remove += atomic.LoadInt64(x.structTotal)
		x.parent = nil
		XXXAddToTotal(s, -remove)
	}

	f.Header = value.header

	f.Ptr = unsafe.Pointer(value)
	XXXAddToTotal(s, atomic.LoadInt64(value.structTotal))
	s.fields[fieldNum] = f
	return nil
}

func MustSetStruct(s *Struct, fieldNum uint16, value *Struct) {
	err := SetStruct(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteStruct deletes a Struct field and updates our storage total.
func DeleteStruct(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil
	}

	if f.Ptr == nil {
		XXXAddToTotal(s, -8)
		return nil
	}

	x := (*Struct)(f.Ptr)
	x.parent = nil
	XXXAddToTotal(s, -atomic.LoadInt64(x.structTotal))
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// GetListBool returns a list of bools at fieldNum.
func GetListBool(s *Struct, fieldNum uint16) (*Bools, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBools); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil, nil
	}

	ptr := (*Bools)(f.Ptr)

	return ptr, nil
}

func MustGetListBool(s *Struct, fieldNum uint16) *Bools {
	b, err := GetListBool(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListBool(s *Struct, fieldNum uint16, value *Bools) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBools); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header != nil { // We had a previous value stored.
		ptr := (*Bools)(f.Ptr)
		XXXAddToTotal(s, -len(ptr.data))
	}

	f.Header = value.data[:8]
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f
	value.s = s
	XXXAddToTotal(s, len(value.data))
	return nil
}
func MustSetListBool(s *Struct, fieldNum uint16, value *Bools) {
	err := SetListBool(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteListBools deletes a list of bools field and updates our storage total.
func DeleteListBools(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBools); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil
	}

	f.Header = nil
	ptr := (*Bools)(f.Ptr)
	XXXAddToTotal(s, -len(ptr.data))
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// GetListNumber returns a list of numbers at fieldNum.
func GetListNumber[N Number](s *Struct, fieldNum uint16) (*Numbers[N], error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return nil, err
	}
	desc := s.mapping.Fields[fieldNum]

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil, nil
	}

	_, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return nil, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	ptr := (*Numbers[N])(f.Ptr)

	return ptr, nil
}

func MustGetListNumber[N Number](s *Struct, fieldNum uint16) *Numbers[N] {
	b, err := GetListNumber[N](s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListNumber[N Number](s *Struct, fieldNum uint16, value *Numbers[N]) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping.Fields[fieldNum]

	_, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum]
	if f.Header != nil { // We had a previous value stored.
		ptr := (*Numbers[N])(f.Ptr)
		XXXAddToTotal(s, -len(ptr.data))
	}

	f.Header = value.data[:8]
	f.Header.SetFieldNum(fieldNum)
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f
	value.s = s
	XXXAddToTotal(s, len(value.data))
	return nil
}

func MustSetListNumber[N Number](s *Struct, fieldNum uint16, value *Numbers[N]) {
	err := SetListNumber(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteListNumber deletes a list of numbers field and updates our storage total.
func DeleteListNumber[N Number](s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.NumericListTypes...); err != nil {
		return err
	}
	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil
	}
	desc := s.mapping.Fields[fieldNum]

	size, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error deleting field number %d: %w", fieldNum, err)
	}

	ptr := (*Numbers[N])(f.Ptr)
	ptr.s = nil

	reduceBy := int(f.Header.Final40()) + int(size) + 8
	XXXAddToTotal(s, -reduceBy)

	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// GetListStruct returns a list of Structs at fieldNum.
func GetListStruct(s *Struct, fieldNum uint16) (*Structs, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum]
	if f.Header == nil { // The zero value
		return nil, nil
	}

	x := (*Structs)(f.Ptr)
	return x, nil
}

func MustGetListStruct(s *Struct, fieldNum uint16) *Structs {
	l, err := GetListStruct(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return l
}

// SetListStructs deletes all existing values and puts in the passed value.
func SetListStructs(s *Struct, fieldNum uint16, value *Structs) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return err
	}

	value.zeroTypeCompression = s.zeroTypeCompression
	for _, v := range value.data {
		v.parent = s
		v.zeroTypeCompression = s.zeroTypeCompression
	}

	if value.Len() > maxDataSize {
		return fmt.Errorf("cannot have more than %d items in a list", maxDataSize)
	}

	if err := DeleteListStructs(s, fieldNum); err != nil {
		return err
	}

	XXXAddToTotal(s, atomic.LoadInt64(value.size))
	f := s.fields[fieldNum]
	f.Header = value.header
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f
	return nil
}

func MustSetListStruct(s *Struct, fieldNum uint16, value *Structs) {
	if err := SetListStructs(s, fieldNum, value); err != nil {
		panic(err)
	}
}

// AppendListStruct adds the values to the list of Structs at fieldNum. Existing items will be retained.
func AppendListStruct(s *Struct, fieldNum uint16, values ...*Struct) error {
	if len(values) == 0 {
		return fmt.Errorf("must add at least a single value")
	}
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return err
	}
	f := s.fields[fieldNum]

	// The list of structs hasn't been created yet, so create it.
	if f.Header == nil {
		fd := s.mapping.Fields[fieldNum]
		var l *Structs
		if fd.SelfReferential {
			l = NewStructs(s.mapping)
		} else {
			l = NewStructs(fd.Mapping)
		}
		f.Ptr = unsafe.Pointer(l)
		XXXAddToTotal(s, 8) // Add the header size of Structs to our parent Struct
	}

	l := (*Structs)(f.Ptr)
	l.s = s
	l.zeroTypeCompression = s.zeroTypeCompression

	if len(values)+l.Len() > maxDataSize {
		return fmt.Errorf("cannot have more than %d items in a list", maxDataSize)
	}

	for _, v := range values {
		if v == nil {
			return fmt.Errorf("cannot pass a nil *Struct")
		}
	}

	if err := l.Append(values...); err != nil {
		return err
	}
	l.header.SetFieldNum(fieldNum)
	l.header.SetFieldType(field.FTListStructs)
	f.Header = l.header

	f.Ptr = unsafe.Pointer(l)
	s.fields[fieldNum] = f

	l.s = s
	return nil
}

func MustAppendListStruct(s *Struct, fieldNum uint16, values ...*Struct) {
	err := AppendListStruct(s, fieldNum, values...)
	if err != nil {
		panic(err)
	}
}

// DeleteListStructs deletes a list of Structs field and updates our storage total.
func DeleteListStructs(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil
	}
	x := (*Structs)(f.Ptr)
	XXXAddToTotal(s, atomic.LoadInt64(x.size))
	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// GetListBytes returns a list of bytes at fieldNum.
func GetListBytes(s *Struct, fieldNum uint16) (*Bytes, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil, nil
	}

	ptr := (*Bytes)(f.Ptr)

	return ptr, nil
}

func MustGetListBytes(s *Struct, fieldNum uint16) *Bytes {
	b, err := GetListBytes(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListBytes(s *Struct, fieldNum uint16, value *Bytes) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return err
	}
	value.s = s

	f := s.fields[fieldNum]
	if f.Header == nil {
		value.header.SetFieldNum(fieldNum)
		f.Header = value.header
		f.Ptr = unsafe.Pointer(value)
		s.fields[fieldNum] = f
		XXXAddToTotal(s, value.dataSize+value.padding+8)
		return nil
	}
	f.Header = value.data[0]
	f.Header.SetFieldNum(fieldNum)
	ptr := (*Bytes)(f.Ptr)
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f

	XXXAddToTotal(s, value.dataSize-ptr.dataSize+value.padding-ptr.padding+8)
	return nil
}

func MustSetListBytes(s *Struct, fieldNum uint16, value *Bytes) {
	if err := SetListBytes(s, fieldNum, value); err != nil {
		panic(err)
	}
}

func MustSetListStrings(s *Struct, fieldNum uint16, value *Strings) {
	if err := SetListBytes(s, fieldNum, value.Bytes()); err != nil {
		panic(err)
	}
}

// DeleteListBytes deletes a list of bytes field and updates our storage total.
func DeleteListBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return err
	}

	f := s.fields[fieldNum]
	if f.Header == nil {
		return nil
	}

	ptr := (*Bytes)(f.Ptr)
	XXXAddToTotal(s, -(ptr.dataSize + ptr.padding))

	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f
	return nil
}

// SetField sets the field value at fieldNum to value. If value isn't valid for that field,
// this will panic.
func SetField(s *Struct, fieldNum uint16, value any) {
	if int(fieldNum) > len(s.fields) {
		panic(fmt.Sprintf("fieldNum %d is invalid", fieldNum))
	}

	switch t := s.mapping.Fields[int(fieldNum)].Type; t {
	case field.FTBool:
		v := value.(bool)
		MustSetBool(s, fieldNum, v)
	case field.FTInt8:
		v := value.(int8)
		MustSetNumber(s, fieldNum, v)
	case field.FTInt16:
		v := value.(int16)
		MustSetNumber(s, fieldNum, v)
	case field.FTInt32:
		v := value.(int32)
		MustSetNumber(s, fieldNum, v)
	case field.FTInt64:
		v := value.(int64)
		MustSetNumber(s, fieldNum, v)
	case field.FTUint8:
		v := value.(uint8)
		MustSetNumber(s, fieldNum, v)
	case field.FTUint16:
		v := value.(uint16)
		MustSetNumber(s, fieldNum, v)
	case field.FTUint32:
		v := value.(uint32)
		MustSetNumber(s, fieldNum, v)
	case field.FTUint64:
		v := value.(uint64)
		MustSetNumber(s, fieldNum, v)
	case field.FTFloat32:
		v := value.(float32)
		MustSetNumber(s, fieldNum, v)
	case field.FTFloat64:
		v := value.(float64)
		MustSetNumber(s, fieldNum, v)
	case field.FTBytes:
		v := value.([]byte)
		MustSetBytes(s, fieldNum, v, false)
	case field.FTString:
		v := value.(string)
		MustSetBytes(s, fieldNum, conversions.UnsafeGetBytes(v), true)
	case field.FTStruct:
		v := value.(*Struct)
		MustSetStruct(s, fieldNum, v)
	case field.FTListBools:
		v := value.(*Bools)
		MustSetListBool(s, fieldNum, v)
	case field.FTListInt8:
		v := value.(*Numbers[int8])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListInt16:
		v := value.(*Numbers[int16])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListInt32:
		v := value.(*Numbers[int32])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListInt64:
		v := value.(*Numbers[int64])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint8:
		v := value.(*Numbers[uint8])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint16:
		v := value.(*Numbers[uint16])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint32:
		v := value.(*Numbers[uint32])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint64:
		v := value.(*Numbers[uint64])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListFloat32:
		v := value.(*Numbers[float32])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListFloat64:
		v := value.(*Numbers[float64])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListBytes:
		v := value.(*Bytes)
		MustSetListBytes(s, fieldNum, v)
	case field.FTListStrings:
		v := value.(*Strings)
		MustSetListStrings(s, fieldNum, v)
	case field.FTListStructs:
		v := value.(*Structs)
		MustSetListStruct(s, fieldNum, v)
	default:
		panic(fmt.Sprintf("bug: unsupported type %T", t))
	}
}

// DeleteField will delete the field entry for fieldNum.
func DeleteField(s *Struct, fieldNum uint16) {
	if int(fieldNum) > len(s.fields) {
		panic(fmt.Sprintf("fieldNum %d is invalid", fieldNum))
	}

	switch t := s.mapping.Fields[int(fieldNum)].Type; t {
	case field.FTBool:
		DeleteBool(s, fieldNum)
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64, field.FTUint8,
		field.FTUint16, field.FTUint32, field.FTUint64, field.FTFloat32, field.FTFloat64:
		DeleteNumber(s, fieldNum)
	case field.FTBytes, field.FTString:
		DeleteBytes(s, fieldNum)
	case field.FTStruct:
		DeleteStruct(s, fieldNum)
	case field.FTListBools:
		DeleteListBools(s, fieldNum)
	case field.FTListInt8:
		DeleteListNumber[int8](s, fieldNum)
	case field.FTListInt16:
		DeleteListNumber[int16](s, fieldNum)
	case field.FTListInt32:
		DeleteListNumber[int32](s, fieldNum)
	case field.FTListInt64:
		DeleteListNumber[int64](s, fieldNum)
	case field.FTListUint8:
		DeleteListNumber[uint8](s, fieldNum)
	case field.FTListUint16:
		DeleteListNumber[uint16](s, fieldNum)
	case field.FTListUint32:
		DeleteListNumber[uint32](s, fieldNum)
	case field.FTListUint64:
		DeleteListNumber[uint64](s, fieldNum)
	case field.FTListFloat32:
		DeleteListNumber[float32](s, fieldNum)
	case field.FTListFloat64:
		DeleteListNumber[float64](s, fieldNum)
	case field.FTListBytes:
		DeleteListBytes(s, fieldNum)
	case field.FTListStrings:
		DeleteListBytes(s, fieldNum)
	case field.FTListStructs:
		DeleteListStructs(s, fieldNum)
	default:
		panic(fmt.Sprintf("bug: unsupported type %T", t))
	}
}

// XXXAddToTotal is used to increment the sizes of everything in a struct by some value.
func XXXAddToTotal[N int64 | int | uint | uint64](s *Struct, value N) {
	if s == nil {
		return
	}
	v := atomic.AddInt64(s.structTotal, int64(value))
	s.header.SetFinal40(uint64(v))
	var ptr = s.parent
	for {
		if ptr == nil {
			return
		}
		v := atomic.AddInt64(ptr.structTotal, int64(value))
		ptr.header.SetFinal40(uint64(v))
		ptr = ptr.parent
	}
}

// validateFieldNum will validate that the type is described in the mapping.Map,
// and if len(ftypes) != 0, that the ftype and mapping.Map[fieldNum].Type are the same.
func validateFieldNum(fieldNum uint16, maps *mapping.Map, ftypes ...field.Type) error {
	if int(fieldNum) >= len(maps.Fields) {
		return fmt.Errorf("fieldNum %d is >= the number of possible fields (%d)", fieldNum, len(maps.Fields))
	}
	if len(ftypes) == 0 {
		return nil
	}

	desc := maps.Fields[fieldNum]
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

func numberToDescCheck[N Number](desc *mapping.FieldDescr) (size uint8, isFloat bool, err error) {
	var t N
	switch any(t).(type) {
	case uint8:
		switch desc.Type {
		case field.FTUint8, field.FTListUint8:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint8 or []uint8 type, was %v", desc.Type)
		}
		size = 8
	case uint16:
		switch desc.Type {
		case field.FTUint16, field.FTListUint16:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint16 or []uint16 type, was %v", desc.Type)
		}
		size = 16
	case uint32:
		switch desc.Type {
		case field.FTUint32, field.FTListUint32:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint32 or []uint32 type, was %v", desc.Type)
		}
		size = 32
	case uint64:
		switch desc.Type {
		case field.FTUint64, field.FTListUint64:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint64 or []uint64 type, was %v", desc.Type)
		}
		size = 64
	case int8:
		switch desc.Type {
		case field.FTInt8, field.FTListInt8:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int8 or []int8 type, was %v", desc.Type)
		}
		size = 8
	case int16:
		switch desc.Type {
		case field.FTInt16, field.FTListInt16:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int16 or []int16 type, was %v", desc.Type)
		}
		size = 16
	case int32:
		switch desc.Type {
		case field.FTInt32, field.FTListInt32:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int32 or []int32 type, was %v", desc.Type)
		}
		size = 32
	case int64:
		switch desc.Type {
		case field.FTInt64, field.FTListInt64:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int64 or []int64 type, was %v", desc.Type)
		}
		size = 64
	case float32:
		switch desc.Type {
		case field.FTFloat32, field.FTListFloat32:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a float32 or []float32 type, was %v", desc.Type)
		}
		size = 32
		isFloat = true
	case float64:
		switch desc.Type {
		case field.FTFloat64, field.FTListFloat64:
		default:
			return 0, false, fmt.Errorf("fieldNum is not a float64 or []float64 type, was %v", desc.Type)
		}
		size = 64
		isFloat = true
	default:
		return 0, false, fmt.Errorf("passed a number value of %T that we do not support", t)
	}
	return size, isFloat, nil
}
