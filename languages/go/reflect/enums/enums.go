// Package enums is for internal use only.
package enums

import (
	"github.com/bearlytools/claw/languages/go/internal/pragma"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
)

type doNotImplement pragma.DoNotImplement

// EnumGroupImpl implements EnumGroup.
type EnumGroupImpl struct {
	doNotImplement

	// GroupName is the name of the EnumGroup.
	GroupName string
	// GroupLen is how many enumerated values are in this group.
	GroupLen int
	// EnumSize is the bit size, 8 or 16, that the values are.
	EnumSize uint8
	// Descrs hold the valuye descriptors.
	Descrs []interfaces.Enum
}

// Name is the name of the enum group.
func (e EnumGroupImpl) Name() string {
	return e.GroupName
}

// Len reports the number of enum values.
func (e EnumGroupImpl) Len() int {
	return e.GroupLen
}

// Get returns the ith EnumValue. It panics if out of bounds.
func (e EnumGroupImpl) Get(i uint16) interfaces.Enum {
	return e.Descrs[i]
}

// ByName returns the EnumValue for an enum named s.
// It returns nil if not found.
func (e EnumGroupImpl) ByName(s string) interfaces.Enum {
	// Enums are usually small and reflection is the slow path. For now,
	// I'm going to simply use a for loop for what I think will be the majority of
	// cases. Go's map implementation is pretty gretat, but I think this will be
	// similar in speed for the majority of cases and not cost us another map allocation.
	for _, descr := range e.Descrs {
		if descr.Name() == s {
			return descr
		}
	}
	return nil
}

func (e EnumGroupImpl) ByValue(i uint16) interfaces.Enum {
	for _, descr := range e.Descrs {
		if descr.Number() == uint16(i) {
			return descr
		}
	}
	return nil
}

// Size returns the size in bits of the enumerator.
func (e EnumGroupImpl) Size() uint8 {
	return e.EnumSize
}

// EnumGroupsImpl implements reflect.EnumGroups.
type EnumGroupsImpl struct {
	doNotImplement

	List   []interfaces.EnumGroup
	Lookup map[string]interfaces.EnumGroup
}

// Len reports the number of enum types.
func (e EnumGroupsImpl) Len() int {
	return len(e.List)
}

// Get returns the ith EnumDescriptor. It panics if out of bounds.
func (e EnumGroupsImpl) Get(i int) interfaces.EnumGroup {
	return e.List[i]
}

// ByName returns the EnumDescriptor for an enum named s.
// It returns nil if not found.
func (e EnumGroupsImpl) ByName(s string) interfaces.EnumGroup {
	return e.Lookup[s]
}

// EnumImpl implements EnumValueDescr.
type EnumImpl struct {
	doNotImplement

	EnumName   string
	EnumNumber uint16
	EnumSize   uint8
}

// Name returns the name of the Enum value.
func (e EnumImpl) Name() string {
	return e.EnumName
}

// Number returns the enum number value.
func (e EnumImpl) Number() uint16 {
	return e.EnumNumber
}

// Size returns the size of the value, either 8 or 16 bits.
func (e EnumImpl) Size() uint8 {
	return e.EnumSize
}
