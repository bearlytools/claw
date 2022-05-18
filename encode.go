package claw

// This file holds all of our encoders which are used to encode our values
// into the various holders of []byte data. These can be read with Get*()
// functions.

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/johnsiilver/claw/internal/binary"
	"golang.org/x/exp/constraints"
)

// EncodeScalar encodes the Scalar value S.
func EncodeScalar[S Scalar](m Marks, field uint16, s S) error {
	switch v := any(s).(type) {
	case bool:
		return encodeBool(m, field, v)
	case int8:
		return encodeInt(m, field, v)
	case int16:
		return encodeInt(m, field, v)
	case int32:
		return encodeInt(m, field, v)
	case int64:
		return encodeInt(m, field, v)
	case uint8:
		return encodeInt(m, field, v)
	case uint16:
		return encodeInt(m, field, v)
	case uint32:
		return encodeInt(m, field, v)
	case uint64:
		return encodeInt(m, field, v)
	case float32:
		return encodeFloat(m, field, v)
	case float64:
		return encodeFloat(m, field, v)
	case string:
		return encodeSlice(m, field, v)
	case []byte:
		return encodeSlice(m, field, v)
	}
	panic(fmt.Sprintf("%T is a Scalar that isn't supported, meaning its a bug", s))
}

func encodeBool(m Marks, field uint16, b bool) error {
	panic("not supported yet")
}

// encodeSlice encodes eiter a string or []byte.
func encodeSlice[D DataSlice](m Marks, field uint16, d D) error {
	if !correctFieldScalar[D](m, field, d) {
		return ErrType
	}

	t := m.mapping[field].Type

	var sb []byte
	switch v := any(d).(type) {
	case string:
		// This gets the []byte used by the string without making a copy. These bytes should
		// not be modified.
		sb = unsafeGetBytes(v)
	case []byte:
		sb = v
	default:
		panic(fmt.Sprintf("encodeSlice got %T, which is not supported but is in DataSlice, so bug"))
	}

	if len(sb) > 4095*_1MiB {
		return fmt.Errorf("cannot set a string larger thatn 4095 MiB")
	}

	var h scalarHolder
	i, ok := m.fields[field]
	if !ok {
		h = scalarHolder{
			header: pool.get64(),
		}
	} else {
		h = i.(scalarHolder)
	}

	// Attach our header which is the field number + field type + the size of the value in bytes.
	binary.Put[uint16](h.header[0:2], field)
	h.header[3] = uint8(t)
	binary.Put[uint32](h.header[3:7], uint32(len(sb)))

	h.data = sb

	m.fields[field] = h
	return nil
}

func encodeInt[V constraints.Integer](m Marks, field uint16, i V) error {
	if !correctFieldScalar[V](m, field, i) {
		return ErrType
	}

	var h scalarHolder
	inter, ok := m.fields[field]
	if !ok {
		h = scalarHolder{
			header: pool.get64(),
		}
	} else {
		h = inter.(scalarHolder)
	}

	// Header info
	binary.Put[uint16](h.header[0:2], field)
	h.header[2] = uint8(m.mapping[field].Type)

	switch any(i).(type) {
	case int8, uint8:
		h.header[3] = uint8(i)
	case int16, uint16:
		binary.Put[uint16](h.header[3:5], uint16(i))
	case int32, uint32:
		binary.Put[uint32](h.header[3:7], uint32(i))
	case int64, uint64:
		h.data = pool.get64()
		binary.Put[uint64](h.data, uint64(i))
	default:
		panic(fmt.Sprintf("encodeInt somehow got %T", i))
	}
	m.fields[field] = h
	return nil
}

func encodeFloat[V constraints.Float](m Marks, field uint16, f V) error {
	if !correctFieldScalar[V](m, field, f) {
		return ErrType
	}

	var h scalarHolder
	inter, ok := m.fields[field]
	if !ok {
		h = scalarHolder{
			header: pool.get64(),
		}
	} else {
		h = inter.(scalarHolder)
	}

	// Header info
	binary.Put[uint16](h.header[0:2], field)
	h.header[2] = uint8(m.mapping[field].Type)

	switch v := any(f).(type) {
	case float32:
		binary.Put[uint32](h.header[3:7], math.Float32bits(v))
	case float64:
		h.data = pool.get64()
		binary.Put[uint64](h.data, math.Float64bits(v))
	}
	m.fields[field] = h
	return nil
}

// EncodeList encodes a list of items.
func EncodeList[L ListItem](m Marks, field uint16, l []L) error {
	items := len(l)
	switch {
	case items == 0: // This also protects the unsafe.Sizeof(v[0]) below from a panic.
		if h, ok := m.fields[field]; ok {
			h.decom()
			delete(m.fields, field)
		}
		return nil
	case items > math.MaxUint32:
		return ErrMaxSliceLen
	}

	if !correctFieldList(m, field, l) {
		return ErrType
	}

	switch v := any(l).(type) {
	case []int8:
		encodeNumList(m, field, sizeOf(v), l)
	case []int16:
		encodeNumList(m, field, sizeOf(v), l)
	case []int32:
		encodeNumList(m, field, sizeOf(v), l)
	case []int64:
		encodeNumList(m, field, sizeOf(v), l)
	case []uint8:
		encodeNumList(m, field, sizeOf(v), l)
	case []uint16:
		encodeNumList(m, field, sizeOf(v), l)
	case []uint32:
		encodeNumList(m, field, sizeOf(v), l)
	case []uint64:
		encodeNumList(m, field, sizeOf(v), l)
	case []float32:
		encodeNumList(m, field, sizeOf(v), l)
	case []float64:
		encodeNumList(m, field, sizeOf(v), l)
	case []string:
		encodeDataSliceList(m, field, l)
	case [][]byte:
		encodeDataSliceList(m, field, l)
	case []Marks:
		panic("not supported yet")
		//encodeStruct(m, field, l)
	default:
		panic(fmt.Sprintf("EncodeList called with type %T, which is not supported but is in the constraints, bug...", l))
	}
	return nil
}

type numbers interface {
	[]int8 | []int16 | []int32 | []int64 | []uint8 | []uint16 | []uint32 | []uint64 | []float32 | []float64
}

func sizeOf[N Number](n []N) int8 {
	return int8(unsafe.Sizeof(n[0]))
}

// encodeNumList encodes a list of numeric values. size is the amount of bytes each value takes.
func encodeNumList[L ListItem](m Marks, field uint16, size int8, l []L) {
	items := len(l)

	// Calculate our storate size.
	dataSize := items * int(size)

	// If we haven't already allocated a value, get a buffer from the pool,
	// extend its length to the maximum capacity of the internal array.
	// If it is not big enough to hold our list, put it back in the pool
	// and allocate the exact size we need.
	var nh numericSliceHolder
	h, ok := m.fields[field]
	if !ok {
		nh.header = pool.get64()
		nh.data = pool.getBuff()
	}else{
		nh = h.(numericSliceHolder)
	}

	nh.data = nh.data[0:cap(nh.data)]
	if len(nh.data) < dataSize {
		pool.put(nh.data)
		nh.data = make([]byte, dataSize)
	} else {
		nh.data = nh.data[0:dataSize]
	}

	// Add our headers which include:
	// * field number
	// * number of items
	// * size of items in bytes
	enc.PutUint16(nh.header[0:2], field)
	switch size {
	case 1:
		nh.header[3] = byte(FTList8)
	case 2:
		nh.header[3] = byte(FTList16)
	case 4:
		nh.header[3] = byte(FTList32)
	case 8:
		nh.header[3] = byte(FTList64)
	}

	binary.Put[uint32](nh.header[4:8], uint32(items))

	// Loop through all our values and encode them.
	index := 0
	for _, n := range l {
		switch v := any(n).(type) {
		case int8:
			nh.data[index] = byte(uint(v))
			index += 1
		case int16:
			enc.PutUint16(nh.data[index:], uint16(v))
			index += 2
		case int32:
			enc.PutUint32(nh.data[index:], uint32(v))
			index += 4
		case int64:
			enc.PutUint64(nh.data[index:], uint64(v))
			index += 8
		case uint8:
			nh.data[index] = byte(v)
			index += 1
		case uint16:
			enc.PutUint16(nh.data[index:], v)
			index += 2
		case uint32:
			enc.PutUint32(nh.data[index:], v)
			index += 4
		case uint64:
			enc.PutUint64(nh.data[index:], v)
			index += 8
		case float32:
			enc.PutUint32(nh.data[index:], math.Float32bits(v))
			index += 4
		case float64:
			enc.PutUint64(nh.data[index:], math.Float64bits(v))
			index += 8
		default:
			panic(fmt.Sprintf("forgot to add support for %T", v))
		}
	}
	m.fields[field] = nh
}

func encodeDataSliceList[L ListItem](m Marks, field uint16, l []L) {
	items := len(l)
	t := m.mapping[field].ListType.Type

	var h dataSliceHolder
	i, ok := m.fields[field]
	if !ok {
		h = dataSliceHolder{header: pool.get64()}
	} else {
		h = i.(dataSliceHolder)
	}
	h.data = make([]dataSliceItemHolder, 0, items)

	// Write list header.
	enc.PutUint16(h.header[0:2], field)
	h.header[3] = byte(FTListBytes)
	enc.PutUint32(h.header[4:8], uint32(items))

	switch t {
	case FTString:
		for _, item := range l {
			v := (any(item)).(string)
			sb := unsafeGetBytes(v)
			dItem := dataSliceItemHolder{header: pool.get32(), data: sb}
			enc.PutUint32(dItem.header, uint32(len(sb)))
			h.data = append(h.data, dItem)
		}
	case FTBytes:
		for _, item := range l {
			v := (any(item)).([]byte)
			dItem := dataSliceItemHolder{header: pool.get32(), data: v}
			enc.PutUint32(dItem.header, uint32(len(v)))
			h.data = append(h.data, dItem)
		}
	}

	m.fields[field] = h
}
