package mapping

import "github.com/bearlytools/claw/internal/field"

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

// Mqp is a map of field numbers to field descriptions.
type Map []*FieldDesc
