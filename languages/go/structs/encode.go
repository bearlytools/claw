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
		if v.Header == nil {
			log.Printf("field %d was skipped for encode", i)
			continue
		}

		if v.Header.FieldNum() != uint16(i) {
			return written, fmt.Errorf("bug: field %d in the index had field number %d(%s), which is a bug", i, v.Header.FieldNum(), v.Header.FieldType())
		}

		desc := s.mapping.Fields[i]
		log.Printf("field %d was: %s", i, desc.Type)

		switch desc.Type {
		// This handles any basic scalar type.
		case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8,
			field.FTUint16, field.FTUint32, field.FTFloat32:
			if s.zeroTypeCompression {
				if v.Header.Final40() == 0 {
					break
				}
			}
			i, err := w.Write(v.Header)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTInt64, field.FTUint64, field.FTFloat64:
			var b *[]byte
			if v.Ptr != nil {
				b = (*[]byte)(v.Ptr)
			}
			if s.zeroTypeCompression {
				if b == nil {
					break
				}
				allZero := true
				for _, u := range *b {
					if u != 0 {
						allZero = false
						break
					}
				}
				if allZero {
					break
				}
			}
			i, err := w.Write(v.Header)
			written += i
			if err != nil {
				return written, err
			}
			i, err = w.Write(*b)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTString, field.FTBytes:
			if s.zeroTypeCompression {
				if v.Header.Final40() == 0 {
					break
				}
			}
			i, err := w.Write(v.Header)
			log.Println("wrote bytes header of: ", i)
			written += i
			if err != nil {
				return written, err
			}
			if v.Ptr == nil {
				break
			}
			b := (*[]byte)(v.Ptr)
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
			value := (*Struct)(v.Ptr)
			log.Printf("the struct ptr: %+#v", value)
			log.Println("struct's fieldNum: ", value.header.FieldNum())
			i, err := value.Marshal(w)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListBools:
			b := (*Bools)(v.Ptr)
			if b.Len() == 0 {
				break
			}
			i, err := w.Write(b.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListInt8:
			x := (*Numbers[int8])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint8:
			x := (*Numbers[uint8])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListInt16:
			x := (*Numbers[int16])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint16:
			x := (*Numbers[uint16])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				written += i
				return written, err
			}
		case field.FTListInt32:
			x := (*Numbers[int32])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint32:
			x := (*Numbers[uint32])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListFloat32:
			x := (*Numbers[float32])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListInt64:
			x := (*Numbers[int64])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListUint64:
			x := (*Numbers[uint64])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListFloat64:
			x := (*Numbers[float64])(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := w.Write(x.Encode())
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListBytes:
			x := (*Bytes)(v.Ptr)
			if x.Len() == 0 {
				break
			}
			i, err := x.Encode(w)
			written += i
			if err != nil {
				return written, err
			}
		case field.FTListStructs:
			x := (*Structs)(v.Ptr)
			if x.Len() == 0 {
				break
			}
			log.Println("before encode: ", written)
			n, err := x.Encode(w)
			written += n
			if err != nil {
				return written, err
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
