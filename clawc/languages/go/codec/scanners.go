package codec

import (
	"github.com/bearlytools/claw/clawc/internal/binary"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/structs"
	"github.com/bearlytools/claw/clawc/languages/go/structs/header"
)

func init() {
	mapping.RegisterScanSizers = registerScanSizers
}

func registerScanSizers(m *mapping.Map) {
	m.ScanSizers = make([]mapping.ScanSizeFunc, len(m.Fields))
	for i, f := range m.Fields {
		m.ScanSizers[i] = scanSizerForType(f.Type)
	}
}

func scanSizerForType(t field.Type) mapping.ScanSizeFunc {
	switch t {
	case field.FTBool:
		return scanSizeScalar8
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		return scanSizeScalar8
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		return scanSizeScalar16
	case field.FTString, field.FTBytes:
		return scanSizeBytes
	case field.FTStruct:
		return scanSizeStruct
	case field.FTListBools:
		return scanSizeListBools
	case field.FTListInt8, field.FTListUint8:
		return scanSizeListInt8
	case field.FTListInt16, field.FTListUint16:
		return scanSizeListInt16
	case field.FTListInt32, field.FTListUint32, field.FTListFloat32:
		return scanSizeListInt32
	case field.FTListInt64, field.FTListUint64, field.FTListFloat64:
		return scanSizeListInt64
	case field.FTListBytes, field.FTListStrings:
		return scanSizeListBytes
	case field.FTListStructs:
		return scanSizeListStructs
	default:
		return scanSizeUnknown
	}
}

// scanSizeScalar8 returns size for scalar types stored entirely in header (8 bytes).
func scanSizeScalar8(data []byte, hdr []byte) uint32 {
	return 8
}

// scanSizeScalar16 returns size for 64-bit scalar types (header + 8 bytes data).
func scanSizeScalar16(data []byte, hdr []byte) uint32 {
	return 16
}

// scanSizeBytes returns size for string/bytes fields.
func scanSizeBytes(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	dataSize := h.Final40()
	return uint32(8 + structs.SizeWithPadding(dataSize))
}

// scanSizeStruct returns size for nested struct fields.
func scanSizeStruct(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	return uint32(h.Final40())
}

// scanSizeListBools returns size for list of booleans.
func scanSizeListBools(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	items := h.Final40()
	wordsNeeded := (items / 64) + 1
	return uint32(8 + (wordsNeeded * 8))
}

// scanSizeListInt8 returns size for list of int8/uint8.
func scanSizeListInt8(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	items := h.Final40()
	return uint32(8 + structs.SizeWithPadding(items))
}

// scanSizeListInt16 returns size for list of int16/uint16.
func scanSizeListInt16(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	items := h.Final40()
	return uint32(8 + structs.SizeWithPadding(items*2))
}

// scanSizeListInt32 returns size for list of int32/uint32/float32.
func scanSizeListInt32(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	items := h.Final40()
	return uint32(8 + structs.SizeWithPadding(items*4))
}

// scanSizeListInt64 returns size for list of int64/uint64/float64.
func scanSizeListInt64(data []byte, hdr []byte) uint32 {
	h := header.Generic(hdr)
	items := h.Final40()
	return uint32(8 + structs.SizeWithPadding(items*8))
}

// scanSizeListBytes returns size for list of bytes/strings field.
// This requires scanning through each entry to calculate total size.
func scanSizeListBytes(data []byte, hdr []byte) uint32 {
	if len(data) < 8 {
		return 0
	}
	h := header.Generic(hdr)
	numItems := h.Final40()
	if numItems == 0 {
		return 8
	}

	size := 8 // header
	remaining := data[8:]

	for i := uint64(0); i < numItems; i++ {
		if len(remaining) < 4 {
			return uint32(size)
		}
		itemSize := int(binary.Get[uint32](remaining[:4]))
		size += 4 + itemSize
		if len(remaining) >= 4+itemSize {
			remaining = remaining[4+itemSize:]
		} else {
			break
		}
	}

	// Add padding
	paddingNeeded := structs.PaddingNeeded(size)
	return uint32(size + paddingNeeded)
}

// scanSizeListStructs returns size for list of structs field.
// This requires scanning through each struct to calculate total size.
func scanSizeListStructs(data []byte, hdr []byte) uint32 {
	if len(data) < 8 {
		return 0
	}
	h := header.Generic(hdr)
	numItems := h.Final40()
	if numItems == 0 {
		return 8
	}

	size := 8 // header
	remaining := data[8:]

	for i := uint64(0); i < numItems; i++ {
		if len(remaining) < 8 {
			return uint32(size)
		}
		structHeader := header.Generic(remaining[:8])
		structSize := int(structHeader.Final40())
		size += structSize
		if len(remaining) >= structSize {
			remaining = remaining[structSize:]
		} else {
			break
		}
	}

	return uint32(size)
}

// scanSizeUnknown handles unknown field types.
func scanSizeUnknown(data []byte, hdr []byte) uint32 {
	return 0
}
