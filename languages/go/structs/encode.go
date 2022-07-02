package structs

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"

	"github.com/bearlytools/claw/internal/field"
)

// Marshal writes out the Struct to an io.Writer.
func (s *Struct) Marshal(w io.Writer) (n int, err error) {
	total := atomic.LoadInt64(s.structTotal)
	if total%8 != 0 {
		return 0, fmt.Errorf("Struct has an internal size(%d) that is not divisible by 8, something is bugged", total)
	}

	if uint64(total) != s.header.Final40() {
		return 0, fmt.Errorf("Struct had internal size(%d), but header size as %d", total, s.header.Final40())
	}
	defer log.Println("Marshal headers says the size is: ", s.header.Final40())
	defer log.Println("Marshal also says the total is: ", total)
	written, err := w.Write(s.header)
	if err != nil {
		return written, err
	}

	for i, v := range s.fields {
		if v.header == nil {
			log.Printf("field %d was skipped for encode", i+1)
			continue
		}

		desc := s.mapping[i]
		log.Printf("field %d was: %s", i+1, desc.Type)

		switch desc.Type {
		// This handles any basic scalar type.
		case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64, field.FTUint8,
			field.FTUint16, field.FTUint32, field.FTUint64, field.FTFloat32, field.FTFloat64:
			i, err := w.Write(v.header)
			written += i
			if err != nil {
				return written, err
			}
			if v.ptr == nil {
				break
			}
			b := (*[]byte)(v.ptr)
			i, err = w.Write(*b)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTString, field.FTBytes:
			i, err := w.Write(v.header)
			log.Println("wrote bytes header of: ", i)
			written += i
			if err != nil {
				return written, err
			}
			if v.ptr == nil {
				break
			}
			b := (*[]byte)(v.ptr)
			i, err = w.Write(*b)
			log.Println("wrote bytes data of: ", i)
			written += i
			if err != nil {
				return written, err
			}
			pad := PaddingNeeded(written)
			i, err = w.Write(Padding(pad))
			log.Println("wrote bytes padding of: ", i)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTStruct:
			log.Println("encoding a Struct")
			value := (*Struct)(v.ptr)
			log.Printf("the struct ptr: %+#v", value)
			log.Println("struct's fieldNum: ", value.header.FieldNum())
			i, err := value.Marshal(w)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListBools:
			b := (*Bool)(v.ptr)
			i, err := w.Write(b.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListInt8:
			x := (*Number[int8])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint8:
			x := (*Number[uint8])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListInt16:
			x := (*Number[int16])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint16:
			x := (*Number[uint16])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				written += i
				return written, err
			}
		case field.FTListInt32:
			x := (*Number[int32])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint32:
			x := (*Number[uint32])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListFloat32:
			x := (*Number[float32])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListInt64:
			x := (*Number[int64])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint64:
			x := (*Number[uint64])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListFloat64:
			x := (*Number[float64])(v.ptr)
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListBytes:
			x := (*Bytes)(v.ptr)
			i, err := x.Encode(w)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListStructs:
			x := (*[]*Struct)(v.ptr)
			n, err := w.Write(v.header)
			written += n
			if err != nil {
				return written, err
			}
			for index, item := range *x {
				item.header.SetFieldNum(uint16(index))
				log.Println("item index: ", item.header.FieldNum())
				n, err := item.Marshal(w)
				written += n
				if err != nil {
					return written, err
				}
			}
		default:
			return written, fmt.Errorf("received a field type %v that we don't support", desc.Type)
		}
	}
	log.Println("wrote: ", written)
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
