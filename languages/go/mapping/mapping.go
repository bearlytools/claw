// Package mapping holds metadata mapping information used to map Struct field numbers to descriptions
// of the fields so that they can be encoded/decoded properly. THIS FILE IS FOR INTERNAL USE ONLY and
// is exposed simply to allow generated packages access.
package mapping

import (
	"fmt"

	"github.com/bearlytools/claw/internal/field"
)

// FieldDescr describes a field.
type FieldDescr struct {
	// Name is the name of the field as described in the .claw file.
	Name string
	// Type is the type of field.
	Type field.Type
	// FieldNum is the field number in the Struct.
	FieldNum uint16
	// IsEnum indicates if the field is an enumerated type. This can only be true
	// if the Type is FTUint8 or FTUint16
	IsEnum bool

	// SelfReferential indicates if an FTStruct or FTListStruct is the same as the containing Struct.
	// If true, Mapping is not set.
	SelfReferential bool
	// Mapping is provided if .Type == FTStruct || FTListStruct. This will describe the Structs fields.
	Mapping *Map
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
