package testing

/*
This is in another package because IDL rendered files import reflect, so doing this in
reflect has a circular dependency problem.
*/

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
	cars "github.com/bearlytools/test_claw_imports/cars/claw"
)

type fieldWant struct {
	// Name is the name of the field as described in the .claw file.
	Name string
	// Type is the type of field.
	Type field.Type
	// FieldNum is the field number inside the Struct.
	FieldNum uint16
	// IsEnum indicates if this field is an enumerator.
	IsEnum bool
	// EnumGroup returns the EnumGroup associated with this field.
	// This is only valid if IsEnum() is true.
	EnumGroup enumGroupWant
	// ItemType returns the name of a Struct if the field is a list of Struct values.
	// If not, this panics.
	ItemType string
}

type enumGroupWant struct {
	// Name is the name of the EnumGroup.
	Name string
	// Len reports the number of enum values.
	Len int
	// Size returns the size in bits of the enumerator.
	Size uint8 // Either 8 or 16
}

func (f fieldWant) Compare(got interfaces.FieldDescr) string {
	b := strings.Builder{}

	if f.Name != got.Name() {
		b.WriteString(fmt.Sprintf("-Name: %s\n+Name: %s\n", got.Name(), f.Name))
	}
	if f.Type != got.Type() {
		b.WriteString(fmt.Sprintf("-Type: %s\n+Type: %s\n", got.Type(), f.Type))
	}
	if f.FieldNum != got.FieldNum() {
		b.WriteString(fmt.Sprintf("-FieldNum: %d\n+FieldNum: %d\n", got.FieldNum(), f.FieldNum))
	}
	if f.IsEnum != got.IsEnum() {
		b.WriteString(fmt.Sprintf("-IsEnum: %v\n+IsEnum: %v\n", got.IsEnum(), f.IsEnum))
	}
	if f.IsEnum && got.IsEnum() {
		log.Printf("EnumGroup:\n%#+v", got)
		if f.EnumGroup.Name != got.EnumGroup().Name() {
			b.WriteString(fmt.Sprintf("-EnumGroup().Name(): %s\n+EnumGroup.Name: %s\n", got.EnumGroup().Name(), f.EnumGroup.Name))
		}
		if f.EnumGroup.Len != got.EnumGroup().Len() {
			b.WriteString(fmt.Sprintf("-EnumGroup().Len(): %v\n+EnumGroup.Len: %v\n", got.EnumGroup().Len(), f.EnumGroup.Len))
		}
		if f.EnumGroup.Size != got.EnumGroup().Size() {
			b.WriteString(fmt.Sprintf("-EnumGroup().Size(): %v\n+EnumGroup.Size: %v\n", got.EnumGroup().Size(), f.EnumGroup.Size))
		}
	}
	if f.ItemType != got.ItemType() {
		b.WriteString(fmt.Sprintf("-ItemType: %s\n+ItemType: %s\n", got.ItemType(), f.ItemType))
	}
	return b.String()
}

func TestGetStructDecr(t *testing.T) {
	vehiclesWant := []fieldWant{
		{
			Name:     "Type",
			Type:     field.FTUint8,
			FieldNum: 0,
			IsEnum:   true,
			EnumGroup: enumGroupWant{
				Name: "Type",
				Len:  3,
				Size: 8,
			},
		},
		{
			Name:     "Cars",
			Type:     field.FTStruct,
			FieldNum: 1,
		},
		{
			Name:     "Trucks",
			Type:     field.FTListStructs,
			FieldNum: 1,
		},
	}

	car := cars.NewCar()
	car.SetYear(2010)
	car.SetManufacturer(manufacturers.Toyota)
	car.SetModel(cars.Vienza)

	v := vehicles.NewVehicle()
	v.SetType(vehicles.Car)
	v.SetCar(car)

	cs := v.ClawStruct()
	for i, f := range cs.Descriptor().Fields() {
		log.Println("field: ", i)
		if diff := vehiclesWant[i].Compare(f); diff != "" {
			t.Errorf("TestGetStructDecr: -want/+got:\n%s", diff)
		}
	}

	/*
		type Struct interface {
			doNotImplement

			// Descriptor returns message descriptor, which contains only the protobuf
			// type information for the message.
			Descriptor() StructDescr

			// New returns a newly allocated and mutable empty Struct.
			New() Struct

			// Range iterates over every populated field in an undefined order,
			// calling f for each field descriptor and value encountered.
			// Range returns immediately if f returns false.
			// While iterating, mutating operations may only be performed
			// on the current field descriptor.
			Range(f func(FieldDescr, Value) bool)

			// Has reports whether a field is populated. This always works for list type fields
			// and Struct fields. With scalar values, this can be interpreted in two ways. If
			// NoZeroValueCompression is on, then this will report if the value has been set or not.
			// If it hasn't, this will report true for all scalar values, string and bytes types, as
			// there is no way to determine if the zero value was set.
			Has(FieldDescr) bool

			// Clear clears the field such that a subsequent Has call reports false.
			//
			// Clearing an extension field clears both the extension type and value
			// associated with the given field number.
			//
			// Clear is a mutating operation and unsafe for concurrent use.
			Clear(FieldDescr)

			// Get retrieves the value for a field.
			//
			// For unpopulated scalars, it returns the default value, where
			// the default value of a bytes scalar is guaranteed to be a copy.
			// For unpopulated composite types, it returns an empty, read-only view
			// of the value; to obtain a mutable reference, use Mutable.
			Get(FieldDescr) Value

			// Set stores the value for a field.
			//
			// For a field belonging to a oneof, it implicitly clears any other field
			// that may be currently set within the same oneof.
			// For extension fields, it implicitly stores the provided ExtensionType.
			// When setting a composite type, it is unspecified whether the stored value
			// aliases the source's memory in any way. If the composite value is an
			// empty, read-only value, then it panics.
			//
			// Set is a mutating operation and unsafe for concurrent use.
			Set(FieldDescr, Value)

			// NewField returns a new value that is assignable to the field
			// for the given descriptor. For scalars, this returns the default value.
			// For lists and Structs, this returns a new, empty, mutable value.
			NewField(FieldDescr) Value

			realType() *structs.Struct
		}
	*/
}
