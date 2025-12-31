// Package mapping holds metadata mapping information used to map Struct field numbers to descriptions
// of the fields so that they can be encoded/decoded properly. THIS FILE IS FOR INTERNAL USE ONLY and
// is exposed simply to allow generated packages access.
package mapping

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/bearlytools/claw/clawc/languages/go/field"
)

// EncodeFunc is the signature for field encoder functions.
// Parameters:
//   - w: destination writer
//   - header: the field header (8 bytes)
//   - ptr: pointer to the field data (may be nil for scalar types stored in header)
//   - desc: field descriptor with type and metadata
//
// Returns bytes written and any error.
// Note: Zero-value compression is always enabled - scalar zero values are skipped.
type EncodeFunc func(w io.Writer, header []byte, ptr unsafe.Pointer, desc *FieldDescr) (int, error)

// ScanSizeFunc calculates the size of a field for offset scanning.
// Parameters:
//   - data: the raw bytes starting at this field
//   - header: the field header (first 8 bytes of data)
//
// Returns the total size of this field in bytes.
type ScanSizeFunc func(data []byte, header []byte) uint32

// LazyDecodeFunc decodes a single field from raw bytes into a struct.
// Parameters:
//   - structPtr: pointer to the Struct being decoded (as unsafe.Pointer to avoid import cycle)
//   - fieldNum: the field number being decoded
//   - data: the raw bytes for this field (includes header)
//   - desc: field descriptor with type and metadata
type LazyDecodeFunc func(structPtr unsafe.Pointer, fieldNum uint16, data []byte, desc *FieldDescr)

// Registration functions - set by the codec/segment packages during init.
// This pattern avoids circular dependencies between mapping and codec/segment.
var (
	// RegisterEncoders is called by Init() to populate the Encoders slice.
	RegisterEncoders func(m *Map)
	// RegisterScanSizers is called by Init() to populate the ScanSizers slice.
	RegisterScanSizers func(m *Map)
	// RegisterLazyDecoders is called by Init() to populate the LazyDecoders slice.
	RegisterLazyDecoders func(m *Map)
	// RegisterSegmentPool is called by Init() to register a per-mapping segment pool.
	RegisterSegmentPool func(m *Map)
)

// FieldDescr describes a field. FieldDescr are created when the IDL renders to a file
// and are generated from information in the idl.File and idl.Struct types.
type FieldDescr struct {
	// Name is the name of the field as described in the .claw file.
	Name string
	// Type is the type of field.
	Type field.Type
	// FieldNum is the field number in the Struct.
	FieldNum uint16
	// StructName is the name of the struct type if Type == FTStruct.
	// This will be either the name of the Struct in this file or [package].[group].
	StructName string
	// IsEnum indicates if the field is an enumerated type. This can only be true
	// if the Type is FTUint8 or FTUint16
	IsEnum bool
	// EnumGroup is the name of the enumeration group this belongs to. This will be
	// either the name of the group in this file or the [package].[group].
	EnumGroup string
	// Package is the package name.
	Package string
	// FullPath is the path to the package as defined by the import statement.
	FullPath string
	// SelfReferential indicates if an FTStruct or FTListStruct is the same as the containing Struct.
	// If true, Mapping is not set.
	SelfReferential bool
	// Mapping is provided if .Type == FTStruct || FTListStruct. This will describe the Structs fields.
	Mapping *Map

	// Map-specific fields (only set when Type == FTMap)
	// IsMap indicates if this field is a map type.
	IsMap bool
	// KeyType is the type of the map key.
	KeyType field.Type
	// ValueType is the type of the map value.
	ValueType field.Type
	// ValueMapping is provided if ValueType == FTStruct. This describes the value Struct's fields.
	ValueMapping *Map
}

func (f *FieldDescr) Validate() error {
	switch f.Type {
	case field.FTListStructs, field.FTStruct:
		if f.Mapping == nil {
			return fmt.Errorf(".%s: type was %v, but had Mapping == nil", f.Name, f.Type)
		}
		if err := f.Mapping.validate(); err != nil {
			return fmt.Errorf(".%s%w", f.Name, err)
		}
	case field.FTMap:
		if !f.IsMap {
			return fmt.Errorf(".%s: type was FTMap but IsMap was false", f.Name)
		}
		if !field.IsValidMapKeyType(f.KeyType) {
			return fmt.Errorf(".%s: invalid map key type %v", f.Name, f.KeyType)
		}
		if f.ValueType == field.FTStruct && f.ValueMapping == nil {
			return fmt.Errorf(".%s: map value type is struct but ValueMapping == nil", f.Name)
		}
		if f.ValueMapping != nil {
			if err := f.ValueMapping.validate(); err != nil {
				return fmt.Errorf(".%s%w", f.Name, err)
			}
		}
	}
	return nil
}

// Map is a map of field numbers to field descriptions for a Struct.
type Map struct {
	// Name of the Struct.
	Name string
	// Pkg is the package the Struct is in.
	Pkg string
	// Path is the path to the package.
	Path string
	// Fields are the field descriptions for all fields in the Struct.
	Fields []*FieldDescr

	// Function pointer tables - initialized once by Init(), used for O(1) dispatch.
	// These replace the type switches in encode/decode hot paths.
	Encoders     []EncodeFunc     // For Marshal()
	ScanSizers   []ScanSizeFunc   // For scanFieldOffsets()
	LazyDecoders []LazyDecodeFunc // For decodeFieldFromRaw()

	initialized bool
}

// Init initializes the function pointer tables for this Map.
// This should be called once during package init for generated code.
// Safe to call multiple times (idempotent).
func (m *Map) Init() {
	if m.initialized {
		return
	}

	// Use registration functions set by the codec/segment packages
	if RegisterEncoders != nil {
		RegisterEncoders(m)
	}
	if RegisterScanSizers != nil {
		RegisterScanSizers(m)
	}
	if RegisterLazyDecoders != nil {
		RegisterLazyDecoders(m)
	}
	if RegisterSegmentPool != nil {
		RegisterSegmentPool(m)
	}

	m.initialized = true

	// Recursively init nested struct mappings
	for _, f := range m.Fields {
		if f.Mapping != nil && f.Mapping != m { // avoid self-reference loop
			f.Mapping.Init()
		}
		// Also init value mappings for map fields
		if f.ValueMapping != nil && f.ValueMapping != m {
			f.ValueMapping.Init()
		}
	}
}

func (m Map) validate() error {
	for _, entry := range m.Fields {
		if err := entry.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ByName retrieves the FieldDesc by name. If the name can't be found, it panics.
func (m Map) ByName(name string) *FieldDescr {
	for _, f := range m.Fields {
		if f.Name == name {
			return f
		}
	}
	panic(fmt.Sprintf("could not find name %q", name))
}

func (m Map) MustValidate() {
	if err := m.validate(); err != nil {
		panic(err)
	}
}
