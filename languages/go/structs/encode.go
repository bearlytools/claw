package structs

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
)

// Marshal writes out the Struct to an io.Writer.
func (s *Struct) Marshal(w io.Writer) (n int, err error) {
	// FAST PATH: If nothing was modified and we have raw data, just write it directly.
	// This avoids re-encoding when the struct was unmarshaled and never changed.
	if !s.modified && s.rawData != nil {
		return w.Write(s.rawData)
	}

	// SLOW PATH: Re-encode the struct from decoded fields.
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

	// Use function pointer dispatch instead of type switch for O(1) dispatch
	encoders := s.mapping.Encoders
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

		// O(1) function pointer dispatch instead of O(N) type switch
		if encoders != nil && encoders[i] != nil {
			n, err := encoders[i](w, v.Header, v.Ptr, desc, s.zeroTypeCompression)
			written += n
			if err != nil {
				return written, err
			}
		} else {
			// Fallback to type switch for backward compatibility
			n, err := s.encodeFieldFallback(w, v, desc)
			written += n
			if err != nil {
				return written, err
			}
		}
	}
	log.Println("wrote: ", written)
	if written != int(total) {
		return written, fmt.Errorf("bug: we wrote %d data out, which is not the same as the total bytes it should take (%d)", written, total)
	}

	return written, nil
}

// encodeFieldFallback is the fallback encoder using type switch.
// Used when function pointers are not initialized (backward compatibility).
func (s *Struct) encodeFieldFallback(w io.Writer, v StructField, desc *mapping.FieldDescr) (int, error) {
	written := 0

	switch desc.Type {
	case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8,
		field.FTUint16, field.FTUint32, field.FTFloat32:
		if s.zeroTypeCompression && v.Header.Final40() == 0 {
			return 0, nil
		}
		return w.Write(v.Header)
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		var b *[]byte
		if v.Ptr != nil {
			b = (*[]byte)(v.Ptr)
		}
		if s.zeroTypeCompression {
			if b == nil {
				return 0, nil
			}
			allZero := true
			for _, u := range *b {
				if u != 0 {
					allZero = false
					break
				}
			}
			if allZero {
				return 0, nil
			}
		}
		n, err := w.Write(v.Header)
		written += n
		if err != nil {
			return written, err
		}
		n, err = w.Write(*b)
		written += n
		return written, err
	case field.FTString, field.FTBytes:
		if s.zeroTypeCompression && v.Header.Final40() == 0 {
			return 0, nil
		}
		n, err := w.Write(v.Header)
		written += n
		if err != nil {
			return written, err
		}
		if v.Ptr == nil {
			return written, nil
		}
		b := (*[]byte)(v.Ptr)
		n, err = w.Write(*b)
		written += n
		if err != nil {
			return written, err
		}
		pad := PaddingNeeded(len(*b))
		n, err = w.Write(Padding(pad))
		written += n
		return written, err
	case field.FTStruct:
		value := (*Struct)(v.Ptr)
		return value.Marshal(w)
	case field.FTListBools:
		b := (*Bools)(v.Ptr)
		if b.Len() == 0 {
			return 0, nil
		}
		return w.Write(b.Encode())
	case field.FTListInt8:
		x := (*Numbers[int8])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListUint8:
		x := (*Numbers[uint8])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListInt16:
		x := (*Numbers[int16])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListUint16:
		x := (*Numbers[uint16])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListInt32:
		x := (*Numbers[int32])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListUint32:
		x := (*Numbers[uint32])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListFloat32:
		x := (*Numbers[float32])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListInt64:
		x := (*Numbers[int64])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListUint64:
		x := (*Numbers[uint64])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListFloat64:
		x := (*Numbers[float64])(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return w.Write(x.Encode())
	case field.FTListBytes:
		x := (*Bytes)(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return x.Encode(w)
	case field.FTListStructs:
		x := (*Structs)(v.Ptr)
		if x.Len() == 0 {
			return 0, nil
		}
		return x.Encode(w)
	default:
		return 0, fmt.Errorf("received a field type %v that we don't support", desc.Type)
	}
}
