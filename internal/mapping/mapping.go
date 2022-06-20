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
	ListType *ListDesc
	// Mapping is provided if .Type == FTStruct. This will describe the
	// Structs fields.
	Mapping Map
}

// ListDesc describes what the entries will be like if the type is a List*.
type ListDesc struct {
	// Type is the type of field in the list.
	Type field.Type
	// Mapping is provided only if Type == FTListStruct.
	Mapping Map
}

// Mqp is a map of field numbers to field descriptions.
type Map []*FieldDesc
