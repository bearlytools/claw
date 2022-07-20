package reflect

import (
	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/languages/go/reflect/internal/value"
	"github.com/bearlytools/claw/languages/go/structs"
)

// ClawStruct indicates that the type represents a Claw Struct.
type ClawStruct interface {
	// ClawReflect returns the reflect.Struct for a Claw Struct.
	ClawReflect() Struct
}

// PackageDescr is used to describe a claw package and its contents.
type PackageDescr = value.PackageDescr

// StructDescrs gives access to the descriptions of a package's struct objects.
type StructDescrs = value.StructDescr

// EnumGroup describes a single set of enum values defined in a claw package.
type EnumGroup = value.EnumGroup

// EnumValueDescr describes an enumerated value.
type EnumValueDescr = value.EnumValueDescr

// Enum is the refection interface for a concrete enum value.
type Enum = value.Enum

// EnumGroups describes enum groups in a package.
type EnumGroups = value.EnumGroups

// List provides access to one of Claw's list types from the int family, uint family,
// lists of bytes/string or list of structs.
type List = value.List

// Kind represents the field's kind, which in Claw nomenclature would be its type.
type Kind kind

type kind struct {
	clawType field.Type
}

func (k Kind) IsValid() bool {
	return k.clawType != field.FTUnknown
}

func (k Kind) String() string {
	return k.clawType.String()
}

type XXXPackageDescrImpl = value.PackageDescrImpl
type XXXEnumGroupImpl = value.EnumGroupImpl
type XXXEnumValueDescrImpl = value.EnumValueDescrImpl
type XXXStructDescrImpl = value.StructDescrImpl
type XXXFieldDescrImpl = value.FieldDescrImpl

func XXXNewStruct(v *structs.Struct) Struct {
	return value.XXXNewStruct(v)
}
