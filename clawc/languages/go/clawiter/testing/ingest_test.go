package testing

import (
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/types/list"
	"github.com/kylelemons/godebug/pretty"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
)

func TestIngestRoundTripCar(t *testing.T) {
	tests := []struct {
		name  string
		setup func() cars.Car
	}{
		{
			name: "Success: basic car",
			setup: func() cars.Car {
				return cars.NewCar().
					SetManufacturer(manufacturers.Tesla).
					SetModel(cars.ModelS).
					SetYear(2023)
			},
		},
		{
			name: "Success: empty car",
			setup: func() cars.Car {
				return cars.NewCar()
			},
		},
		{
			name: "Success: Toyota Venza",
			setup: func() cars.Car {
				return cars.NewCar().
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
			},
		},
	}

	for _, test := range tests {
		original := test.setup()

		// Round-trip: Walk -> Ingest
		ingested := cars.NewCar()
		if err := ingested.IngestWithOptions(original.Walk(), clawiter.IngestOptions{}); err != nil {
			t.Errorf("TestIngestRoundTripCar(%s): Ingest error: %s", test.name, err)
			continue
		}

		// Compare field values
		if original.Manufacturer() != ingested.Manufacturer() {
			t.Errorf("TestIngestRoundTripCar(%s): Manufacturer mismatch: got %v, want %v",
				test.name, ingested.Manufacturer(), original.Manufacturer())
		}
		if original.Model() != ingested.Model() {
			t.Errorf("TestIngestRoundTripCar(%s): Model mismatch: got %v, want %v",
				test.name, ingested.Model(), original.Model())
		}
		if original.Year() != ingested.Year() {
			t.Errorf("TestIngestRoundTripCar(%s): Year mismatch: got %v, want %v",
				test.name, ingested.Year(), original.Year())
		}
	}
}

func TestIngestRoundTripVehicle(t *testing.T) {
	tests := []struct {
		name  string
		setup func() vehicles.Vehicle
	}{
		{
			name: "Success: vehicle with car",
			setup: func() vehicles.Vehicle {
				car := cars.NewCar().
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
				return vehicles.NewVehicle().
					SetType(vehicles.Car).
					SetCar(car)
			},
		},
		{
			name: "Success: vehicle with enum list",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle().
					SetType(vehicles.Truck).
					SetTypes(list.NewEnums[vehicles.Type]().Append(vehicles.Car, vehicles.Truck))
			},
		},
		{
			name: "Success: vehicle with bool list",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle().
					SetBools(list.NewBools().Append(true, false, true))
			},
		},
		{
			name: "Success: empty vehicle",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle()
			},
		},
		{
			name: "Success: vehicle with all fields",
			setup: func() vehicles.Vehicle {
				car := cars.NewCar().
					SetManufacturer(manufacturers.Ford).
					SetModel(cars.GT).
					SetYear(2020)
				return vehicles.NewVehicle().
					SetType(vehicles.Car).
					SetCar(car).
					SetTypes(list.NewEnums[vehicles.Type]().Append(vehicles.Car, vehicles.Truck, vehicles.Unknown)).
					SetBools(list.NewBools().Append(true, false))
			},
		},
	}

	for _, test := range tests {
		original := test.setup()

		// Round-trip: Walk -> Ingest
		ingested := vehicles.NewVehicle()
		if err := ingested.IngestWithOptions(original.Walk(), clawiter.IngestOptions{}); err != nil {
			t.Errorf("TestIngestRoundTripVehicle(%s): Ingest error: %s", test.name, err)
			continue
		}

		// Compare Type enum
		if original.Type() != ingested.Type() {
			t.Errorf("TestIngestRoundTripVehicle(%s): Type mismatch: got %v, want %v",
				test.name, ingested.Type(), original.Type())
		}

		// Compare nested Car struct
		origCar := original.Car()
		ingCar := ingested.Car()
		origCarStruct := origCar.XXXGetStruct()
		ingCarStruct := ingCar.XXXGetStruct()
		switch {
		case origCarStruct == nil && ingCarStruct != nil:
			t.Errorf("TestIngestRoundTripVehicle(%s): Car should be nil", test.name)
		case origCarStruct != nil && ingCarStruct == nil:
			t.Errorf("TestIngestRoundTripVehicle(%s): Car should not be nil", test.name)
		case origCarStruct != nil && ingCarStruct != nil:
			if origCar.Manufacturer() != ingCar.Manufacturer() {
				t.Errorf("TestIngestRoundTripVehicle(%s): Car.Manufacturer mismatch", test.name)
			}
			if origCar.Model() != ingCar.Model() {
				t.Errorf("TestIngestRoundTripVehicle(%s): Car.Model mismatch", test.name)
			}
			if origCar.Year() != ingCar.Year() {
				t.Errorf("TestIngestRoundTripVehicle(%s): Car.Year mismatch", test.name)
			}
		}

		// Compare Types enum list
		origTypes := original.Types()
		ingTypes := ingested.Types()
		switch {
		case origTypes.IsNil() && !ingTypes.IsNil():
			t.Errorf("TestIngestRoundTripVehicle(%s): Types should be nil", test.name)
		case !origTypes.IsNil() && ingTypes.IsNil():
			t.Errorf("TestIngestRoundTripVehicle(%s): Types should not be nil", test.name)
		case !origTypes.IsNil() && !ingTypes.IsNil():
			origTypesSlice := origTypes.Slice()
			ingTypesSlice := ingTypes.Slice()
			if diff := pretty.Compare(origTypesSlice, ingTypesSlice); diff != "" {
				t.Errorf("TestIngestRoundTripVehicle(%s): Types mismatch: -want/+got:\n%s",
					test.name, diff)
			}
		}

		// Compare Bools list
		origBools := original.Bools()
		ingBools := ingested.Bools()
		switch {
		case origBools.IsNil() && !ingBools.IsNil():
			t.Errorf("TestIngestRoundTripVehicle(%s): Bools should be nil", test.name)
		case !origBools.IsNil() && ingBools.IsNil():
			t.Errorf("TestIngestRoundTripVehicle(%s): Bools should not be nil", test.name)
		case !origBools.IsNil() && !ingBools.IsNil():
			origBoolsSlice := origBools.Slice()
			ingBoolsSlice := ingBools.Slice()
			if diff := pretty.Compare(origBoolsSlice, ingBoolsSlice); diff != "" {
				t.Errorf("TestIngestRoundTripVehicle(%s): Bools mismatch: -want/+got:\n%s",
					test.name, diff)
			}
		}
	}
}
