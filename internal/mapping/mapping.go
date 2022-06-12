package mapping

import "github.com/bearlytools/claw/internal/field"

// FieldDesc describes a field.
type FieldDesc struct {
	// Type is the type of field.
	Type field.Type

	// MapKeyType describes the map's key type if Type == FTMap.
	MapKeyType *FieldDesc
	// MapValueType describes the map's value type if Type == FTMap.
	MapValueType *FieldDesc
	// ListType describes the list's value type if Type == FTList.
	ListType *FieldDesc
}

// Mqp is a map of field numbers to field descriptions.
type Map []*FieldDesc
