package structs

import (
	"fmt"
	"io"
	"sync/atomic"

	"github.com/bearlytools/claw/internal/field"
)

// Marshal writes out the Struct to an io.Writer.
func (s *Struct) Marshal(w io.Writer) (n int, err error) {
	total := atomic.LoadInt64(s.total)
	if total%8 != 0 {
		return 0, fmt.Errorf("Struct has an internal size(%d) that is not divisible by 8, something is bugged", total)
	}

	s.header.DataSize = uint64(atomic.LoadInt64(s.total))
	err = s.header.validate()
	if err != nil {
		return 0, fmt.Errorf("invalid Struct: %w", err)
	}

	written, err := s.header.Write(w)
	if err != nil {
		return written, err
	}

	mappingSize := len(s.mapping)
	for n, v := range s.fields {
		// This occurs when we ingested a Struct that had fields that our version of the Claw file
		// does not have. We retain this data.
		if n > mappingSize {
			i, err := w.Write(v)
			written += i
			if err != nil {
				return written, err
			}
			continue
		}

		fd := s.mapping[n]
		switch fd.Type {
		// This handles any basic scalar type.
		case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64, field.FTUint8,
			field.FTUint16, field.FTUint32, field.FTUint64, field.FTFloat32, field.FTFloat64, field.FTString,
			field.FTBytes:
			i, err := w.Write(v)
			written += i
			if err != nil {
				return written, err
			}
			written += i
		case field.FTStruct:
			index, ok := s.fieldNumToStruct[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTStruct type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.structs) {
				return written, fmt.Errorf("bug: a Struct field that was a FTStruct type (field %d) did not have a corresponding entry", n)
			}
			v := s.structs[index]
			// v can be nil because no one supplied a struct, so we don't need to write anything.
			if v == nil {
				continue
			}
			if i, err := v.Marshal(w); err != nil {
				written += i
				return written, err
			}
		case field.FTListBool:
			index, ok := s.fieldNumToList[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTListBool type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.lists) {
				return written, fmt.Errorf("bug: a Struct field that was a FTListBool type (field %d) did not have a corresponding entry", n)
			}
			ptr := s.lists[index]
			// v can be nil because no one supplied a value, so we don't need to write anything.
			if ptr == nil {
				continue
			}
			v := (*Bool)(ptr)
			if i, err := w.Write(v.Encode()); err != nil {
				written += i
				return written, err
			}
		case field.FTList8:
			index, ok := s.fieldNumToList[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTList8 type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.lists) {
				return written, fmt.Errorf("bug: a Struct field that was a FTList8 type (field %d) did not have a corresponding entry", n)
			}
			ptr := s.lists[index]
			// v can be nil because no one supplied a value, so we don't need to write anything.
			if ptr == nil {
				continue
			}
			switch fd.ListType.Type {
			case field.FTInt8:
				v := (*Number[int8])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint8:
				v := (*Number[uint8])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List8 field did not specify the item type (FTUint8, FTInt8)")
			}
		case field.FTList16:
			index, ok := s.fieldNumToList[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTList16 type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.lists) {
				return written, fmt.Errorf("bug: a Struct field that was a FTList16 type (field %d) did not have a corresponding entry", n)
			}
			ptr := s.lists[index]
			// v can be nil because no one supplied a value, so we don't need to write anything.
			if ptr == nil {
				continue
			}
			switch fd.ListType.Type {
			case field.FTInt16:
				v := (*Number[int16])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint16:
				v := (*Number[uint16])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List16 field did not specify the item type (FTUint16, FTInt16)")
			}
		case field.FTList32:
			index, ok := s.fieldNumToList[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTList32 type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.lists) {
				return written, fmt.Errorf("bug: a Struct field that was a FTList32 type (field %d) did not have a corresponding entry", n)
			}
			ptr := s.lists[index]
			// v can be nil because no one supplied a value, so we don't need to write anything.
			if ptr == nil {
				continue
			}
			switch fd.ListType.Type {
			case field.FTInt32:
				v := (*Number[int32])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint32:
				v := (*Number[uint32])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTFloat32:
				v := (*Number[float32])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List32 field did not specify the item type (FTUint32, FTInt32, FTFloat32)")
			}
		case field.FTList64:
			index, ok := s.fieldNumToList[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTList64 type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.lists) {
				return written, fmt.Errorf("bug: a Struct field that was a FTList64 type (field %d) did not have a corresponding entry", n)
			}
			ptr := s.lists[index]
			// v can be nil because no one supplied a value, so we don't need to write anything.
			if ptr == nil {
				continue
			}
			switch fd.ListType.Type {
			case field.FTInt64:
				v := (*Number[int64])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint64:
				v := (*Number[uint64])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTFloat64:
				v := (*Number[float64])(ptr)
				if i, err := w.Write(v.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List64 field did not specify the item type (FTUint64, FTInt64, FTFloat64)")
			}
		case field.FTListBytes:
			index, ok := s.fieldNumToList[uint16(n)]
			if !ok {
				return written, fmt.Errorf("bug: a Struct field that was a FTListBytes type was described(field %d) that did not have a mapping", n)
			}
			if index >= len(s.lists) {
				return written, fmt.Errorf("bug: a Struct field that was a FTListBytes type (field %d) did not have a corresponding entry", n)
			}
			ptr := s.lists[index]
			// v can be nil because no one supplied a value, so we don't need to write anything.
			if ptr == nil {
				continue
			}
			v := (*Bytes)(ptr)
			for _, data := range v.Encode() {
				if i, err := w.Write(data); err != nil {
					written += i
					return written, err
				}
			}
		case field.FTListStruct:
			panic("not supported yet")
		default:
			return written, fmt.Errorf("received a field type %v that we don't support", fd.Type)
		}
	}
	if written != int(total) {
		return written, fmt.Errorf("bug: we wrote %d data out, which is not the same as the total bytes it should take (%d)", written, total)
	}

	return written, nil
}

/*
// Encode encodes the Struct into its byte representation.
func (s Struct) Encode(fieldNum uint16) ([]byte, error) {
	buff := bytes.Buffer{}

	for _, f := range s.fields {
		size += len(f)
	}

	for n, v := range s.fields {
		fd := s.mapping[n]

		switch fd.Type {
		case FTBool:

		case FTInt8:
		case FTInt16:
		case FTInt32:
		case FTInt64:
		case FTUint8:
		case FTUint16:
		case FTUint32:
		case FTUint64:
		case FTFloat32:
		case FTFloat64:
		case FTString:
		case FTBytes:
		case FTStruct:
		case FTListBool:
		case FTList8:
		case FTList16:
		case FTList32:
		case FTList64:
		case FTListBytes:
		case FTListStruct:
		default:
		}
	}
}
*/
