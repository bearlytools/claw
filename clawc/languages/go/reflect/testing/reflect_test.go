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

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/reflect"
	"github.com/bearlytools/claw/clawc/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/clawc/languages/go/types/list"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
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
	if got == nil {
		panic("can't Compare(nil)")
	}
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
	if f.Type == field.FTListStructs {
		if f.ItemType != got.ItemType() {
			b.WriteString(fmt.Sprintf("-ItemType: %s\n+ItemType: %s\n", got.ItemType(), f.ItemType))
		}
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
			Name:     "Car",
			Type:     field.FTStruct,
			FieldNum: 1,
		},
		{
			Name:     "Truck",
			Type:     field.FTListStructs,
			FieldNum: 2,
			ItemType: "Truck",
		},
		{
			Name:     "Types",
			Type:     field.FTListUint8,
			FieldNum: 3,
			IsEnum:   true,
			EnumGroup: enumGroupWant{
				Name: "Type",
				Len:  3,
				Size: 8,
			},
		},
		{
			Name:     "Bools",
			Type:     field.FTListBools,
			FieldNum: 4,
		},
		{
			Name:     "Name",
			Type:     field.FTString,
			FieldNum: 5,
		},
		{
			Name:     "VIN",
			Type:     field.FTBytes,
			FieldNum: 6,
		},
	}

	// Setup a vehicle the normal way.
	v := vehicles.NewVehicle().
		SetType(vehicles.Car).
		SetCar(cars.NewCar().
			SetYear(2010).
			SetManufacturer(manufacturers.Toyota).
			SetModel(cars.Venza),
		).
		SetBools(list.NewBools().Append(
			true, false, true,
		),
		).
		SetTypes(
			list.NewEnums[vehicles.Type]().Append(
				vehicles.Car,
				vehicles.Truck,
			),
		)

	// Setup a vehicle the reflect way.
	vehiclesPkgDescr := vehicles.PackageDescr()
	vehicleDescr := vehiclesPkgDescr.Structs().ByName("Vehicle")
	mfgPkgDescr := manufacturers.PackageDescr()
	carsPkgDescr := cars.PackageDescr()
	carDescr := carsPkgDescr.Structs().ByName("Car")
	carValue := carDescr.New()
	carValue.Set(carDescr.FieldDescrByName("Year"), reflect.ValueOfNumber[uint16](2010))

	enumNumber := mfgPkgDescr.Enums().ByName("Manufacturer").ByValue(1).Number()
	carValue.Set(
		carDescr.FieldDescrByName("Manufacturer"),
		reflect.ValueOfEnum(uint8(enumNumber), mfgPkgDescr.Enums().ByName("Manufacturer")),
	)

	enumNumber = carsPkgDescr.Enums().ByName("Model").ByName("Venza").Number()
	carValue.Set(
		carDescr.FieldDescrByName("Model"),
		reflect.ValueOfEnum(uint8(enumNumber), carsPkgDescr.Enums().ByName("Model")),
	)

	vehicleValue := vehicleDescr.New()

	vehicleValue.Set(vehicleDescr.FieldDescrByName("Car"), reflect.ValueOfStruct(carValue))

	log.Printf("yes: %#+v", vehicleValue.Get(vehicleDescr.FieldDescrByName("Car")))
	for x, cs := range []interfaces.Struct{v.ClawStruct(), vehicleValue} {
		csDescr := cs.Descriptor()
		for i, f := range csDescr.Fields() {
			log.Printf("comparing vehiclesWant[%d]", i)
			if diff := vehiclesWant[i].Compare(f); diff != "" {
				if x == 0 {
					t.Errorf("TestGetStructDecr(normalSetup): fieldDescriptors -want/+got:\n%s", diff)
				} else {
					t.Errorf("TestGetStructDecr(reflectSetup): fieldDescriptors -want/+got:\n%s", diff)
				}
			}
		}
		carFD := csDescr.FieldDescrByName("Car")
		carStruct := cs.Get(carFD).Struct()
		yearDescr := carStruct.Descriptor().FieldDescrByName("Year")
		mfgDescr := carStruct.Descriptor().FieldDescrByName("Manufacturer")

		year := carStruct.Get(yearDescr)
		if year.Uint() != 2010 {
			t.Errorf("TestGetStructDecr: could not extract Vehicle.Car.Year: got %d, want %d", year.Uint(), 2010)
		}
		mfg := carStruct.Get(mfgDescr)
		if mfg.Enum().Number() != uint16(manufacturers.Toyota) {
			t.Errorf("TestGetStructDecr: could not extract Vehicle.Car.Manufacturer: got %d, want %d", mfg.Enum(), manufacturers.Toyota)
		}
	}
}
