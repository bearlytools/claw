package claw

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"golang.org/x/exp/constraints"
)

const (
	_1MiB = 1048576
)

var (
	ErrType        = errors.New("incorrect type for field")
	ErrMaxSliceLen = errors.New("maximum size for slice exceeded")
)

var enc = binary.LittleEndian

type Number interface {
	constraints.Integer | constraints.Float
}

// DataSlice is a type constraint that supports strings and byte slices.
type DataSlice interface {
	string | []byte
}

// Scalars is a type constraint that supports all our scalar types.
type Scalar interface {
	bool | DataSlice | Number
}

type ListItem interface {
	int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 | bool | string | []byte | Marks | Scalar | DataSlice
}

// List is a type constraint that supports our concrete types of lists.
type List interface {
	[]int8 | []int16 | []int32 | []int64 | []uint8 | []uint16 | []uint32 | []uint64 | []bool | []string | [][]byte | []Marks
}

// FieldType represents the type of data that is held in a byte field.
type FieldType uint8

const (
	FTUnknown    FieldType = 0
	FTBool                 = 1
	FTInt8                 = 2
	FTInt16                = 3
	FTInt32                = 4
	FTInt64                = 5
	FTUint8                = 6
	FTUint16               = 7
	FTUint32               = 8
	FTUint64               = 9
	FTFloat32              = 10
	FTFloat64              = 11
	FTString               = 12
	FTBytes                = 13
	FTStruct               = 14
	FTListBool             = 15
	FTList8                = 16
	FTList16               = 17
	FTList32               = 18
	FTList64               = 19
	FTListBytes            = 20
	FTListStruct           = 21
)

func isList(ft FieldType) bool {
	switch ft {
	case FTListBool, FTList8, FTList16, FTList32, FTList64, FTListBytes, FTListStruct:
		return true
	}
	return false
}

// FieldDesc describes a field.
type FieldDesc struct {
	// Type is the type of field.
	Type FieldType

	// MapKeyType describes the map's key type if Type == FTMap.
	MapKeyType *FieldDesc
	// MapValueType describes the map's value type if Type == FTMap.
	MapValueType *FieldDesc
	// ListType describes the list's value type if Type == FTList.
	ListType *FieldDesc
}

// Mapping is a mapping of field numbers to field descriptions.
type Mapping map[uint16]*FieldDesc

// pool holds a few sync.Pool(s) that we can use for buffer reuse.
var pool = &pools{
	_32: &sync.Pool{
		New: func() any {
			return make([]byte, 4)
		},
	},
	_64: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 8)
		},
	},
	_128: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 16)
		},
	},
	buff: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 64)
		},
	},
}

type pools struct {
	_32 *sync.Pool
	_64  *sync.Pool
	_128 *sync.Pool
	buff *sync.Pool
}

func (p *pools) get32() []byte {
	return p._32.Get().([]byte)
}

func (p *pools) get64() []byte {
	return p._64.Get().([]byte)
}

func (p *pools) get128() []byte {
	return p._128.Get().([]byte)
}

func (p *pools) getBuff() []byte {
	return p.buff.Get().([]byte)
}

func (p *pools) put(b []byte) {
	switch len(b) {
	case 32:
		p._32.Put(b)
	case 64:
		p._64.Put(b)
	case 128:
		p._128.Put(b)
	default:
		p.buff.Put(b)
	}
}

// dataHolder holds some data of data representing a type.
type dataHolder interface {
	// holder simply indicates this is a dataHolder.
	holder()
	// header returns the data header.
	Header() []byte
	// decom puts all data storage back in our pools for reuse.
	decom()
}

// scalarHolder is used to hold any scalar type.
type scalarHolder struct {
	header []byte
	data   []byte
}

func (s scalarHolder) holder() {}
func (s scalarHolder) Header() []byte {return s.header}
func (s scalarHolder) decom() {
	pool.put(s.header)
	pool.put(s.data)
}

// numericSliceHolder holds a list of numeric values of some size.
type numericSliceHolder struct {
	header []byte
	data []byte
}

func (n numericSliceHolder) holder() {}
func (n numericSliceHolder) Header() []byte{return n.header}
func (n numericSliceHolder) decom() {
	pool.put(n.header)
	pool.put(n.data)
}

// dataSliceHolder is used to hold [][]byte or []string data.
type dataSliceHolder struct {
	header []byte
	data   []dataSliceItemHolder
}

func (d dataSliceHolder) holder() {}
func (d dataSliceHolder) Header() []byte{return d.header}
func (d dataSliceHolder) decom() {
	pool.put(d.header)
	for _, i := range d.data {
		i.decom()
	}
}

// dataSliceItemHolder is an individual string or []byte held in a slice.
type dataSliceItemHolder struct {
	header []byte
	data   []byte
}

func (d dataSliceItemHolder) decom() {
	pool.put(d.header)
	pool.put(d.data)
}

// Marks is used to hold the byte representation of our values that are either going to be encoded
// or have been decoded.
type Marks struct {
	// mapping holds our Mapping object that allows us to understand what field number holds what value type.
	mapping Mapping
	// fields keys on a field number and has values that are the byte representation of the field.
	// encoding can be done simply by writing attaching the header and writing the fields out in any order.
	fields map[uint16]dataHolder
}

// Decomn decoms the Marks by taking all fields and returning them to our allocation pools.
// The Marks instance and fields should no longer be used.
func (m Marks) Decom() {
	for _, f := range m.fields {
		f.decom()
	}
}

/*
func (m Marks) SetInt8(field uint16, value int8) error {
	if m.mapping[field].Type != FTInt8 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get64()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint16(b[2:3], uint16(value))
	m.fields[field] = b
	return nil
}

func (m Marks) GetInt8(field uint16) (int8, error) {
	if m.mapping[field].Type != FTInt8 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := int8(enc.Uint16(b[2:3]))
	return value, nil
}

func (m Marks) SetInt16(field uint16, value int16) error {
	if m.mapping[field].Type != FTInt16 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get64()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint16(b[2:4], uint16(value))
	m.fields[field] = b
	return nil
}

func (m Marks) GetInt16(field uint16) (int16, error) {
	if m.mapping[field].Type != FTInt16 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := int16(enc.Uint16(b[2:4]))
	return value, nil
}

func (m Marks) SetInt32(field uint16, value int32) error {
	if m.mapping[field].Type != FTInt32 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get64()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint32(b[2:6], uint32(value))
	m.fields[field] = b
	return nil
}

func (m Marks) GetInt32(field uint16) (int32, error) {
	if m.mapping[field].Type != FTInt32 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := int32(enc.Uint32(b[2:6]))
	return value, nil
}

func (m Marks) SetInt64(field uint16, value int64) error {
	if m.mapping[field].Type != FTInt64 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get128()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint64(b[2:10], uint64(value))
	m.fields[field] = b
	return nil
}

func (m Marks) GetInt64(field uint16) (int64, error) {
	if m.mapping[field].Type != FTInt64 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := int64(enc.Uint64(b[2:10]))
	return value, nil
}

func (m Marks) SetUint8(field uint16, value uint8) error {
	if m.mapping[field].Type != FTUint8 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get64()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint16(b[2:3], uint16(value))
	m.fields[field] = b
	return nil
}

func (m Marks) GetUint8(field uint16) (uint8, error) {
	if m.mapping[field].Type != FTUint8 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := uint8(enc.Uint16(b[2:3]))
	return value, nil
}

func (m Marks) SetUint16(field uint16, value uint16) error {
	if m.mapping[field].Type != FTUint16 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get64()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint16(b[2:4], value)
	m.fields[field] = b
	return nil
}

func (m Marks) GetUint16(field uint16) (uint16, error) {
	if m.mapping[field].Type != FTUint16 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := uint16(enc.Uint16(b[2:4]))
	return value, nil
}

func (m Marks) SetUint32(field uint16, value uint32) error {
	if m.mapping[field].Type != FTUint32 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = pool.get64()
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint32(b[2:6], value)
	m.fields[field] = b
	return nil
}

func (m Marks) GetUint32(field uint16) (uint32, error) {
	if m.mapping[field].Type != FTUint16 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := enc.Uint32(b[2:6])
	return value, nil
}

func (m Marks) SetUint64(field uint16, value uint64) error {
	if m.mapping[field].Type != FTUint64 {
		return ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		b = make([]byte, 128)
	}
	enc.PutUint16(b[0:2], field)
	enc.PutUint64(b[2:10], value)
	m.fields[field] = b
	return nil
}

func (m Marks) GetUint64(field uint16) (uint64, error) {
	if m.mapping[field].Type != FTUint64 {
		return 0, ErrType
	}
	b, ok := m.fields[field]
	if !ok {
		return 0, nil
	}
	value := uint64(enc.Uint64(b[2:10]))
	return value, nil
}

func (m Marks) SetString(field uint16, value string) error {
	if len(value) > 4095*_1MiB {
		return fmt.Errorf("cannot set a string larger thatn 4095 MiB")
	}
	if m.mapping[field].Type != FTString {
		return ErrType
	}
	b := pool.getBuff()

	// Attach our header which is the field number + the size of the value in bytes.
	enc.PutUint16(b[0:2], field)
	enc.PutUint32(b[2:6], uint32(len(value)))

	// This gets the []byte used by the string without making a copy. These bytes should
	// not be modified.
	sb := unsafeGetBytes(value)

	padding := len(sb) % 8
	b = append(b[8:], sb...)
	if padding > 0 {
		b = append(b[8+len(sb):], make([]byte, padding)...)
	}

	m.fields[field] = b[:8+len(sb)+padding]
	return nil
}

func (m Marks) GetString(field uint16) (string, error) {
	if m.mapping[field].Type != FTString {
		return "", ErrType
	}

	b, ok := m.fields[field]
	if !ok {
		return "", nil
	}

	// This is the length of the data in bytes.
	l := enc.Uint32(b[2:6])

	return byteSlice2String(b[6 : 6+l]), nil
}

func (m Marks) SetBytes(field uint16, value []byte) error {
	if len(value) > 4095*_1MiB {
		return fmt.Errorf("cannot set bytes larger thatn 4095 MiB")
	}
	if m.mapping[field].Type != FTBytes {
		return ErrType
	}
	b := pool.getBuff()

	// Attach our header which is the field number + the size of the value in bytes.
	enc.PutUint16(b[0:2], field)
	enc.PutUint32(b[2:6], uint32(len(value)))

	padding := len(value) % 8
	b = append(b[8:], value...)
	if padding > 0 {
		b = append(b[8+len(value):], make([]byte, padding)...)
	}

	m.fields[field] = b[:8+len(value)+padding]
	return nil
}

func (m Marks) GetBytes(field uint16) ([]byte, error) {
	if m.mapping[field].Type != FTString {
		return nil, ErrType
	}

	b, ok := m.fields[field]
	if !ok {
		return nil, nil
	}

	// This is the length of the data in bytes.
	l := enc.Uint32(b[2:6])

	return b[6 : 6+l], nil
}
*/

func byteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func unsafeGetBytes(s string) []byte {
	return (*[0x7fff0000]byte)(unsafe.Pointer(
		(*reflect.StringHeader)(unsafe.Pointer(&s)).Data),
	)[:len(s):len(s)]
}

func correctFieldList[L ListItem](m Marks, field uint16, l []L) bool {
	e, ok := m.mapping[field]
	if !ok {
		return false
	}

	switch e.Type {
	case FTList8, FTList16, FTList32, FTList64, FTListStruct, FTListBytes:
	default:
		return false
	}
	t := e.ListType.Type

	switch any(l).(type) {
	case []bool:
		if t != FTBool {
			return false
		}
	case []int8:
		if t != FTInt8 {
			return false
		}
	case []int16:
		if t != FTInt16 {
			return false
		}
	case []int32:
		if t != FTInt32 {
			return false
		}
	case []int64:
		if t != FTInt64 {
			return false
		}
	case []uint8:
		if t != FTUint8 {
			return false
		}
	case []uint16:
		if t != FTUint16 {
			return false
		}
	case []uint32:
		if t != FTUint32 {
			return false
		}
	case []uint64:
		if t != FTUint64 {
			return false
		}
	case []string:
		if t != FTString {
			return false
		}
	case [][]byte:
		if t != FTBytes {
			return false
		}
	case []Marks:
		if t != FTStruct {
			return false
		}
	default:
		panic(fmt.Sprintf("correctFieldList received %T, which was not supported but was in the type constraint, bug...."))
	}
	return true
}

func correctFieldScalar[S Scalar](m Marks, field uint16, s S) bool {
	t := m.mapping[field].Type

	switch any(s).(type) {
	case int8:
		if t != FTInt8 {
			return false
		}
	case int16:
		if t != FTInt16 {
			return false
		}
	case int32:
		if t != FTInt32 {
			return false
		}
	case int64:
		if t != FTInt64 {
			return false
		}
	case uint8:
		if t != FTUint8 {
			return false
		}
	case uint16:
		if t != FTUint16 {
			return false
		}
	case uint32:
		if t != FTUint32 {
			return false
		}
	case uint64:
		if t != FTUint64 {
			return false
		}
	case float32:
		if t != FTFloat32 {
			return false
		}
	case float64:
		if t != FTFloat64 {
			return false
		}
	case string:
		if t != FTString {
			return false
		}
	case []byte:
		if t != FTBytes {
			return false
		}
	default:
		panic(fmt.Sprintf("got %T, which apparently is a Scalar with no support", s))
	}
	return true
}
