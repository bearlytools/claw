package mapping

import (
	"fmt"

	"github.com/bearlytools/claw/internal/field"
)

// FieldDesc describes a field.
type FieldDesc struct {
	// Name is the name of the field as described in the .claw file.
	Name string
	// GoName is the name of field, if required.
	GoName string
	// Type is the type of field.
	Type field.Type

	// ListType describes the list's value type if Type == FTList.
	ListType field.Type
	// Mapping is provided if .Type == FTStruct || FTListStruct. This will describe the Structs fields.
	Mapping Map
}

func (f *FieldDesc) Validate() error {
	switch f.Type {
	case field.FTList8, field.FTList16, field.FTList32, field.FTList64:
		switch f.ListType {
		case field.FTUint8, field.FTUint16, field.FTUint32, field.FTUint64, field.FTInt8, field.FTInt16,
			field.FTInt32, field.FTInt64, field.FTFloat32, field.FTFloat64:
		default:
			return fmt.Errorf(".%s: type was %v, but had non numeric .ListType %v", f.Name, f.Type, f.ListType)
		}
	case field.FTListStruct, field.FTStruct:
		if f.Mapping == nil {
			return fmt.Errorf(".%s: type was %v, but had Mapping == nil", f.Name, f.Type)
		}
		if err := f.Mapping.validate(); err != nil {
			return fmt.Errorf(".%s%w", f.Name, err)
		}
	}
	return nil
}

// Mqp is a map of field numbers to field descriptions.
type Map []*FieldDesc

func (m Map) validate() error {
	for _, entry := range m {
		if err := entry.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m Map) MustValidate() {
	if err := m.validate(); err != nil {
		panic(err)
	}
}
