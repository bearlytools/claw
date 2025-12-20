// Package codec provides optimized encode/decode functions for claw serialization.
// It registers function pointers with the mapping package to enable O(1) dispatch
// instead of O(N) type switches in hot paths.
package codec

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/bearlytools/claw/languages/go/structs"
	"github.com/bearlytools/claw/languages/go/structs/header"
)

func init() {
	mapping.RegisterEncoders = registerEncoders
}

func registerEncoders(m *mapping.Map) {
	m.Encoders = make([]mapping.EncodeFunc, len(m.Fields))
	for i, f := range m.Fields {
		m.Encoders[i] = encoderForType(f.Type)
	}
}

func encoderForType(t field.Type) mapping.EncodeFunc {
	switch t {
	case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32,
		field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		return encodeScalar32
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		return encodeScalar64
	case field.FTString, field.FTBytes:
		return encodeBytes
	case field.FTStruct:
		return encodeStruct
	case field.FTListBools:
		return encodeListBools
	case field.FTListInt8:
		return encodeListInt8
	case field.FTListUint8:
		return encodeListUint8
	case field.FTListInt16:
		return encodeListInt16
	case field.FTListUint16:
		return encodeListUint16
	case field.FTListInt32:
		return encodeListInt32
	case field.FTListUint32:
		return encodeListUint32
	case field.FTListFloat32:
		return encodeListFloat32
	case field.FTListInt64:
		return encodeListInt64
	case field.FTListUint64:
		return encodeListUint64
	case field.FTListFloat64:
		return encodeListFloat64
	case field.FTListBytes:
		return encodeListBytes
	case field.FTListStrings:
		return encodeListStrings
	case field.FTListStructs:
		return encodeListStructs
	default:
		return encodeUnsupported
	}
}

// encodeScalar32 encodes scalar types that fit in the 40-bit header value.
func encodeScalar32(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	h := header.Generic(hdr)
	if zeroComp && h.Final40() == 0 {
		return 0, nil
	}
	return w.Write(hdr)
}

// encodeScalar64 encodes 64-bit scalar types (int64, uint64, float64).
func encodeScalar64(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	var b *[]byte
	if ptr != nil {
		b = (*[]byte)(ptr)
	}
	if zeroComp {
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
	written, err := w.Write(hdr)
	if err != nil {
		return written, err
	}
	n, err := w.Write(*b)
	written += n
	return written, err
}

// encodeBytes encodes string and bytes fields.
func encodeBytes(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	h := header.Generic(hdr)
	if zeroComp && h.Final40() == 0 {
		return 0, nil
	}
	written, err := w.Write(hdr)
	if err != nil {
		return written, err
	}
	if ptr == nil {
		return written, nil
	}
	b := (*[]byte)(ptr)
	n, err := w.Write(*b)
	written += n
	if err != nil {
		return written, err
	}
	// Add padding to align to 8 bytes
	pad := structs.PaddingNeeded(len(*b))
	n, err = w.Write(structs.Padding(pad))
	written += n
	return written, err
}

// encodeStruct encodes a nested struct field.
func encodeStruct(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	value := (*structs.Struct)(ptr)
	return value.Marshal(w)
}

// encodeListBools encodes a list of booleans.
func encodeListBools(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	b := (*structs.Bools)(ptr)
	if b.Len() == 0 {
		return 0, nil
	}
	return w.Write(b.Encode())
}

// encodeListInt8 encodes a list of int8.
func encodeListInt8(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[int8])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListUint8 encodes a list of uint8.
func encodeListUint8(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[uint8])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListInt16 encodes a list of int16.
func encodeListInt16(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[int16])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListUint16 encodes a list of uint16.
func encodeListUint16(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[uint16])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListInt32 encodes a list of int32.
func encodeListInt32(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[int32])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListUint32 encodes a list of uint32.
func encodeListUint32(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[uint32])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListFloat32 encodes a list of float32.
func encodeListFloat32(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[float32])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListInt64 encodes a list of int64.
func encodeListInt64(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[int64])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListUint64 encodes a list of uint64.
func encodeListUint64(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[uint64])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListFloat64 encodes a list of float64.
func encodeListFloat64(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Numbers[float64])(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return w.Write(x.Encode())
}

// encodeListBytes encodes a list of byte slices.
func encodeListBytes(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Bytes)(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return x.Encode(w)
}

// encodeListStrings encodes a list of strings (stored as underlying Bytes).
func encodeListStrings(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Strings)(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	// Strings is a wrapper around *Bytes, use the underlying Bytes for encoding
	return x.Bytes().Encode(w)
}

// encodeListStructs encodes a list of structs.
func encodeListStructs(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	x := (*structs.Structs)(ptr)
	if x.Len() == 0 {
		return 0, nil
	}
	return x.Encode(w)
}

// encodeUnsupported handles unsupported field types.
func encodeUnsupported(w io.Writer, hdr []byte, ptr unsafe.Pointer, desc *mapping.FieldDescr, zeroComp bool) (int, error) {
	return 0, fmt.Errorf("unsupported field type %v for encoding", desc.Type)
}
