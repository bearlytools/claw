package claw

import (
	"context"
	"math"
	"fmt"

	"golang.org/x/exp/constraints"
	"github.com/johnsiilver/claw/internal/binary"
)

// This file holds all our getters for reading our in memory structures
// holding []byte data and outputting the correct types for user consumption.

/*
// GetScalar retrieves a Scalar type from a Marks at field.
func GetScalar[S Scalar](m Marks, field uint16) S {
	i, ok := m.fields[field]
	if !ok {
		var result S
		return result
	}
	h := i.(scalarHolder)

	t := m.mapping[field].Type

	var v S
	switch t {
	case FTInt8:
		v = getInteger[int8](m, t, h)
		return v
	case FTInt16:
		return getInteger[int16](m, t, h)
	case FTInt32:
		return getInteger[int32](m, t, h)
	case FTInt64:
		return getInteger[int64](m, t, h)
	case FTUint8:
		return getInteger[uint8](m, t, h)
	case FTUint16:
		return getInteger[uint16](m, t, h)
	case FTUint32:
		return getInteger[uint32](m, t, h)
	case FTUint64:
		return getInteger[uint64](m, t, h)
	case FTFloat32:
		return getFloat[float32](m, t, h)
	case FTFloat64:
		return getFloat[float64](m, t, h)
	case FTString:
		return getSlice[string](m, t, h)
	case FTByte:
		return getSlice[[]byte](m, t, h)
	default:
		panic("GetScalar() used on field type %v, bug...", t)
	}
}
*/

func getSlice[D DataSlice](m Marks, t FieldType, h scalarHolder) D {
	l := enc.Uint32(h.header[4:8]) // The length of the slice
	switch t {
	case FTString:
		return D(byteSlice2String(h.data[0:l]))
	case FTBytes:
		return D(h.data[0:l])
	}
	panic("getSlice received %T which is not supported but in the constraints, bug....")
}

func getInteger[I constraints.Integer](m Marks, h scalarHolder) I {
	return binary.Get[I](h.header)
	/*
	switch t {
	case FTInt8:
		b := h.header[4]
		return I(int8(b))
	case FTInt16:
		return I(binary.Get[int16](h.header[4:6]))
	case FTInt32:
		return I(binary.Get[int32](h.header[4:8]))
	case FTInt64:
		return I(binary.Get[int64](h.data))
	case FTUint8:
		return I(uint8(h.header[4:5]))
	case FTUint16:
		return I(binary.Get[uint16](h.header[4:6]))
	case FTUint32:
		return I(binary.Get[uint32](h.header[4:8]))
	case FTUint64:
		return I(binary.Get[uint64](h.data))
	}
	panic(fmt.Sprintf("getInteger() received field type %v, which is not supported but in the constraint, bug...", t))
	*/
}

func getFloat[F constraints.Float](m Marks, h scalarHolder) F {
	var v F // Only used for type detection
	switch any(v).(type) {
	case float32:
		return F(math.Float32frombits(binary.Get[uint32](h.header[3:7])))
	case float64:
		h.data = pool.get64()
		return F(math.Float64frombits(binary.Get[uint64](h.data)))
	}
	panic(fmt.Sprintf("getFloat() received field type %v, which is not supported but in the constraint, bug...", v))
}

// GetIntSlice returns all integers stored in field if field is a list field. from cannot
// be less than 0 and to cannot be greater than the number of items. You can set to == -1
// to get all items starting with from until the end of items.
// This causes an allocation of a slice to hold the values. For no allocations, use RangeIntSlice().
func GetIntSlice[V constraints.Integer](m Marks, field uint16, from, to int) ([]V, error) {
	items, err := ListSize(m, field)
	if err != nil {
		return nil, err
	}
	if items == 0 {
		return nil, nil
	}

	sl := make([]V, 0, items)

	ch, err := RangeIntSlice[V](context.Background(), m, field, from, to)
	if err != nil {
		return nil, err
	}

	for item := range ch {
		sl = append(sl, item)
	}

	return sl, nil
}

// ListSize retrieves the length of list stored in a field.
func ListSize(m Marks, field uint16) (int, error) {
	t := m.mapping[field].Type
	if !isList(t){
		return 0, ErrType
	}

	i, ok := m.fields[field]
	if !ok {
		return 0, nil
	}

	return int(binary.Get[uint32](i.Header()[4:6])), nil
}

// RangeIntSlice ranges over field representing a list of integers.
func RangeIntSlice[V constraints.Integer](ctx context.Context, m Marks, field uint16, from, to int) (chan V, error) {
	t := m.mapping[field].Type
	if !isList(t){
		return nil, ErrType
	}

	size := 0
	switch t {
	case FTList8:
		size = 1
	case FTList16:
		size = 2
	case FTList32:
		size = 4
	case FTList64:
		size = 8
	default:
		return nil, ErrType
	}

	// If the field doesn't exist, that is fine, we just give them back a closed channel.
	i, ok := m.fields[field]
	if !ok {
		ch := make(chan V)
		close(ch)
		return ch, nil
	}

	nh := i.(numericSliceHolder)

	items := int(binary.Get[uint32](nh.Header()[4:6]))
	if items == 0 {
		return nil, fmt.Errorf("invalid encoding: a list cannot be encoded with 0 items")
	}

	if len(nh.data) != size * items {
		return nil, fmt.Errorf("invalid encoding: data size was %d, data number of items was %d, but data size was %d", size, items, size * items)
	}

	// Bounds check
	switch {
	case from < 0:
		panic("from cannot be set to a negative integer")
	case from >= items:
		panic("from is out of range")
	case to < 0:
		to = items
	case to > items:
		panic("to is out of range")
	case from > to:
		panic("from cannot be < to")
	}

	b := nh.data[size*from : size*to]

	ch := make(chan V, 1)
	go func() {
		defer close(ch)

		for i := 0; i < items; i++ {
			ch <- V(binary.Get[V](b))
			b = b[size:]
		}
	}()
	return ch, nil
}

/*
func intSliceVal[V constraints.Integer](m Marks, field uint16, index int) (V, int) {
	b := m.fields[field] // The field should already be checked that it exists before we get here.
	var v V

	switch m.mapping[field].ListType.Type {
	case FTInt8:
		v = V(int8(b[index]))
		index += 1
	case FTInt16:
		v = V(int16(enc.Uint16(b[index:])))
		index += 2
	case FTInt32:
		v = V(int32(enc.Uint32(b[index:])))
		index += 4
	case FTInt64:
		v = V(int64(enc.Uint64(b[index:])))
		index += 8
	case FTUint8:
		v = V(uint8(b[index]))
		index += 1
	case FTUint16:
		v = V(enc.Uint16(b[index:]))
		index += 2
	case FTUint32:
		v = V(enc.Uint32(b[index:]))
		index += 4
	case FTUint64:
		v = V(enc.Uint64(b[index:]))
		index += 8
	default:
		panic(fmt.Sprintf("forgot to add support for %T", m.mapping[field].ListType.Type))
	}
	return v, index
}
*/
