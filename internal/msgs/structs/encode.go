package structs

import (
	"fmt"
	"io"
	"sync/atomic"

	"github.com/bearlytools/claw/internal/field"
)

// Marshal writes out the Struct to an io.Writer.
func (s *Struct) Marshal(w io.Writer) (n int, err error) {
	total := atomic.LoadInt64(s.structTotal)
	if total%8 != 0 {
		return 0, fmt.Errorf("Struct has an internal size(%d) that is not divisible by 8, something is bugged", total)
	}
	s.header.SetFinal40(uint64(total))

	written, err := w.Write(s.header)
	if err != nil {
		return written, err
	}

	for n, v := range s.fields {
		if v.header == nil {
			continue
		}
		fd := s.mapping[n]
		switch fd.Type {
		// This handles any basic scalar type.
		case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64, field.FTUint8,
			field.FTUint16, field.FTUint32, field.FTUint64, field.FTFloat32, field.FTFloat64, field.FTString,
			field.FTBytes:
			i, err := w.Write(v.header)
			written += i
			if err != nil {
				return written, err
			}
			if v.ptr != nil {
				b := (*[]byte)(v.ptr)
				i, err := w.Write(*b)
				written += i
				if err != nil {
					return written, err
				}
			}
		case field.FTStruct:
			s := (*Struct)(v.ptr)
			if i, err := s.Marshal(w); err != nil {
				written += i
				return written, err
			}
		case field.FTListBool:
			b := (*Bool)(v.ptr)
			if i, err := w.Write(b.Encode()); err != nil {
				written += i
				return written, err
			}
		case field.FTList8:
			switch fd.ListType.Type {
			case field.FTInt8:
				x := (*Number[int8])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint8:
				x := (*Number[uint8])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List8 field did not specify the item type (FTUint8, FTInt8)")
			}
		case field.FTList16:
			switch fd.ListType.Type {
			case field.FTInt16:
				x := (*Number[int16])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint16:
				x := (*Number[uint16])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List16 field did not specify the item type (FTUint16, FTInt16)")
			}
		case field.FTList32:
			switch fd.ListType.Type {
			case field.FTInt32:
				x := (*Number[int32])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint32:
				x := (*Number[uint32])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTFloat32:
				x := (*Number[float32])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List32 field did not specify the item type (FTUint32, FTInt32, FTFloat32)")
			}
		case field.FTList64:
			switch fd.ListType.Type {
			case field.FTInt64:
				x := (*Number[int64])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTUint64:
				x := (*Number[uint64])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			case field.FTFloat64:
				x := (*Number[float64])(v.ptr)
				if i, err := w.Write(x.Encode()); err != nil {
					written += i
					return written, err
				}
			default:
				return written, fmt.Errorf("bug: mapping data for a List64 field did not specify the item type (FTUint64, FTInt64, FTFloat64)")
			}
		case field.FTListBytes:
			x := (*Bytes)(v.ptr)
			for _, data := range x.Encode() {
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