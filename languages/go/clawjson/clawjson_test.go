package clawjson

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
)

func TestMarshalSimpleCar(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func() cars.Car
		options []MarshalOption
		want    string
	}{
		{
			name: "Success: car with enum strings",
			setup: func() cars.Car {
				return cars.NewCar(ctx).
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
			},
			want: `{"Manufacturer":"Toyota","Model":"Venza","Year":2010}`,
		},
		{
			name: "Success: car with enum numbers",
			setup: func() cars.Car {
				return cars.NewCar(ctx).
					SetManufacturer(manufacturers.Tesla).
					SetModel(cars.ModelS).
					SetYear(2023)
			},
			options: []MarshalOption{WithUseEnumNumbers(true)},
			want:    `{"Manufacturer":3,"Model":3,"Year":2023}`,
		},
		{
			name: "Success: empty car (zero values)",
			setup: func() cars.Car {
				return cars.NewCar(ctx)
			},
			want: `{"Manufacturer":"Unknown","Model":"ModelUnknown","Year":0}`,
		},
	}

	for _, test := range tests {
		c := test.setup()
		got, err := Marshal(ctx, c, test.options...)
		switch {
		case err != nil:
			t.Errorf("TestMarshalSimpleCar(%s): got err == %s, want err == nil", test.name, err)
			continue
		}
		if got.String() != test.want {
			t.Errorf("TestMarshalSimpleCar(%s): got %s, want %s", test.name, got.String(), test.want)
		}
		got.Release(ctx)
	}
}

func TestMarshalNestedStruct(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func() vehicles.Vehicle
		options []MarshalOption
		want    string
	}{
		{
			name: "Success: vehicle with nested car",
			setup: func() vehicles.Vehicle {
				car := cars.NewCar(ctx).
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Car).
					SetCar(car)
			},
			want: `{"Type":"Car","Car":{"Manufacturer":"Toyota","Model":"Venza","Year":2010},"Truck":null,"Types":null,"Bools":null}`,
		},
		{
			name: "Success: vehicle with nil car",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Truck)
			},
			want: `{"Type":"Truck","Car":null,"Truck":null,"Types":null,"Bools":null}`,
		},
	}

	for _, test := range tests {
		v := test.setup()
		got, err := Marshal(ctx, v, test.options...)
		switch {
		case err != nil:
			t.Errorf("TestMarshalNestedStruct(%s): got err == %s, want err == nil", test.name, err)
			continue
		}
		if got.String() != test.want {
			t.Errorf("TestMarshalNestedStruct(%s): got %s, want %s", test.name, got.String(), test.want)
		}
		got.Release(ctx)
	}
}

func TestMarshalLists(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func() vehicles.Vehicle
		options []MarshalOption
		want    string
	}{
		{
			name: "Success: vehicle with bool list",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetBools(true, false, true)
			},
			want: `{"Type":"Unknown","Car":null,"Truck":null,"Types":null,"Bools":[true,false,true]}`,
		},
		{
			name: "Success: vehicle with enum list (strings)",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
			want: `{"Type":"Unknown","Car":null,"Truck":null,"Types":["Car","Truck"],"Bools":null}`,
		},
		{
			name: "Success: vehicle with enum list (numbers)",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
			options: []MarshalOption{WithUseEnumNumbers(true)},
			want:    `{"Type":0,"Car":null,"Truck":null,"Types":[1,2],"Bools":null}`,
		},
	}

	for _, test := range tests {
		v := test.setup()
		got, err := Marshal(ctx, v, test.options...)
		switch {
		case err != nil:
			t.Errorf("TestMarshalLists(%s): got err == %s, want err == nil", test.name, err)
			continue
		}
		if got.String() != test.want {
			t.Errorf("TestMarshalLists(%s): got %s, want %s", test.name, got.String(), test.want)
		}
		got.Release(ctx)
	}
}

func TestMarshalWriter(t *testing.T) {
	ctx := context.Background()
	car := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Ford).
		SetModel(cars.GT).
		SetYear(2020)

	var buf bytes.Buffer
	err := MarshalWriter(ctx, car, &buf)
	switch {
	case err != nil:
		t.Errorf("TestMarshalWriter: got err == %s, want err == nil", err)
		return
	}

	want := `{"Manufacturer":"Ford","Model":"GT","Year":2020}`
	if buf.String() != want {
		t.Errorf("TestMarshalWriter: got %s, want %s", buf.String(), want)
	}
}

func TestArray(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func(*Array) error
		options []MarshalOption
		want    string
	}{
		{
			name: "Success: empty array",
			setup: func(a *Array) error {
				return nil
			},
			want: `[]`,
		},
		{
			name: "Success: single element",
			setup: func(a *Array) error {
				car := cars.NewCar(ctx).
					SetManufacturer(manufacturers.Toyota).
					SetYear(2010)
				return a.Write(ctx, car)
			},
			want: `[{"Manufacturer":"Toyota","Model":"ModelUnknown","Year":2010}]`,
		},
		{
			name: "Success: multiple elements",
			setup: func(a *Array) error {
				car1 := cars.NewCar(ctx).SetManufacturer(manufacturers.Toyota).SetYear(2010)
				car2 := cars.NewCar(ctx).SetManufacturer(manufacturers.Tesla).SetYear(2023)
				if err := a.Write(ctx, car1); err != nil {
					return err
				}
				return a.Write(ctx, car2)
			},
			want: `[{"Manufacturer":"Toyota","Model":"ModelUnknown","Year":2010},{"Manufacturer":"Tesla","Model":"ModelUnknown","Year":2023}]`,
		},
		{
			name: "Success: with enum numbers option",
			setup: func(a *Array) error {
				car := cars.NewCar(ctx).SetManufacturer(manufacturers.Ford).SetYear(2015)
				return a.Write(ctx, car)
			},
			options: []MarshalOption{WithUseEnumNumbers(true)},
			want:    `[{"Manufacturer":2,"Model":0,"Year":2015}]`,
		},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		a, err := NewArray(&buf, test.options...)
		if err != nil {
			t.Errorf("TestArray(%s): NewArray error: %s", test.name, err)
			continue
		}
		if err := test.setup(a); err != nil {
			t.Errorf("TestArray(%s): setup error: %s", test.name, err)
			continue
		}
		if err := a.Close(); err != nil {
			t.Errorf("TestArray(%s): Close error: %s", test.name, err)
			continue
		}
		if buf.String() != test.want {
			t.Errorf("TestArray(%s): got %s, want %s", test.name, buf.String(), test.want)
		}
	}
}

func TestMarshalProducesValidJSON(t *testing.T) {
	ctx := context.Background()
	car := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)

	vehicle := vehicles.NewVehicle(ctx).
		SetType(vehicles.Car).
		SetCar(car).
		SetBools(true, false).
		SetTypes(vehicles.Car, vehicles.Truck)

	got, err := Marshal(ctx, vehicle)
	switch {
	case err != nil:
		t.Errorf("TestMarshalProducesValidJSON: Marshal error: %s", err)
		return
	}
	defer got.Release(ctx)

	var parsed map[string]any
	if err := json.Unmarshal(got.Bytes(), &parsed); err != nil {
		t.Errorf("TestMarshalProducesValidJSON: produced invalid JSON: %s\nJSON: %s", err, got.String())
		return
	}

	want := map[string]any{
		"Type": "Car",
		"Car": map[string]any{
			"Manufacturer": "Toyota",
			"Model":        "Venza",
			"Year":         float64(2010),
		},
		"Truck": nil,
		"Types": []any{"Car", "Truck"},
		"Bools": []any{true, false},
	}

	if diff := pretty.Compare(want, parsed); diff != "" {
		t.Errorf("TestMarshalProducesValidJSON: -want/+got:\n%s", diff)
	}
}

func TestUnmarshalRoundTripCar(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func() cars.Car
		options []MarshalOption
	}{
		{
			name: "Success: car with enum strings",
			setup: func() cars.Car {
				return cars.NewCar(ctx).
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
			},
		},
		{
			name: "Success: car with enum numbers",
			setup: func() cars.Car {
				return cars.NewCar(ctx).
					SetManufacturer(manufacturers.Tesla).
					SetModel(cars.ModelS).
					SetYear(2023)
			},
			options: []MarshalOption{WithUseEnumNumbers(true)},
		},
		{
			name: "Success: empty car (zero values)",
			setup: func() cars.Car {
				return cars.NewCar(ctx)
			},
		},
	}

	for _, test := range tests {
		original := test.setup()

		// Marshal to JSON
		jsonData, err := Marshal(ctx, original, test.options...)
		if err != nil {
			t.Errorf("TestUnmarshalRoundTripCar(%s): Marshal error: %s", test.name, err)
			continue
		}

		// Unmarshal back into a new struct
		restored := cars.NewCar(ctx)
		if err := Unmarshal(ctx, jsonData.Bytes(), &restored); err != nil {
			t.Errorf("TestUnmarshalRoundTripCar(%s): Unmarshal error: %s\nJSON: %s", test.name, err, jsonData.String())
			jsonData.Release(ctx)
			continue
		}
		jsonData.Release(ctx)

		// Compare fields
		if original.Manufacturer() != restored.Manufacturer() {
			t.Errorf("TestUnmarshalRoundTripCar(%s): Manufacturer mismatch: got %v, want %v",
				test.name, restored.Manufacturer(), original.Manufacturer())
		}
		if original.Model() != restored.Model() {
			t.Errorf("TestUnmarshalRoundTripCar(%s): Model mismatch: got %v, want %v",
				test.name, restored.Model(), original.Model())
		}
		if original.Year() != restored.Year() {
			t.Errorf("TestUnmarshalRoundTripCar(%s): Year mismatch: got %v, want %v",
				test.name, restored.Year(), original.Year())
		}
	}
}

func TestUnmarshalRoundTripVehicle(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func() vehicles.Vehicle
		options []MarshalOption
	}{
		{
			name: "Success: vehicle with nested car",
			setup: func() vehicles.Vehicle {
				car := cars.NewCar(ctx).
					SetManufacturer(manufacturers.Toyota).
					SetModel(cars.Venza).
					SetYear(2010)
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Car).
					SetCar(car)
			},
		},
		{
			name: "Success: vehicle with bool list",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetBools(true, false, true)
			},
		},
		{
			name: "Success: vehicle with enum list (strings)",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
		},
		{
			name: "Success: vehicle with enum list (numbers)",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
			options: []MarshalOption{WithUseEnumNumbers(true)},
		},
		{
			name: "Success: vehicle with all fields",
			setup: func() vehicles.Vehicle {
				car := cars.NewCar(ctx).
					SetManufacturer(manufacturers.Ford).
					SetModel(cars.GT).
					SetYear(2020)
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Car).
					SetCar(car).
					SetTypes(vehicles.Car, vehicles.Truck).
					SetBools(true, false)
			},
		},
	}

	for _, test := range tests {
		original := test.setup()

		// Marshal to JSON
		jsonData, err := Marshal(ctx, original, test.options...)
		if err != nil {
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Marshal error: %s", test.name, err)
			continue
		}

		// Unmarshal back into a new struct
		restored := vehicles.NewVehicle(ctx)
		if err := Unmarshal(ctx, jsonData.Bytes(), &restored); err != nil {
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Unmarshal error: %s\nJSON: %s", test.name, err, jsonData.String())
			jsonData.Release(ctx)
			continue
		}
		jsonData.Release(ctx)

		// Compare Type enum
		if original.Type() != restored.Type() {
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Type mismatch: got %v, want %v",
				test.name, restored.Type(), original.Type())
		}

		// Compare nested Car struct
		origCar := original.Car()
		resCar := restored.Car()
		origCarStruct := origCar.XXXGetStruct()
		resCarStruct := resCar.XXXGetStruct()
		switch {
		case origCarStruct == nil && resCarStruct != nil:
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Car should be nil", test.name)
		case origCarStruct != nil && resCarStruct == nil:
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Car should not be nil", test.name)
		case origCarStruct != nil && resCarStruct != nil:
			if origCar.Manufacturer() != resCar.Manufacturer() {
				t.Errorf("TestUnmarshalRoundTripVehicle(%s): Car.Manufacturer mismatch", test.name)
			}
			if origCar.Model() != resCar.Model() {
				t.Errorf("TestUnmarshalRoundTripVehicle(%s): Car.Model mismatch", test.name)
			}
			if origCar.Year() != resCar.Year() {
				t.Errorf("TestUnmarshalRoundTripVehicle(%s): Car.Year mismatch", test.name)
			}
		}

		// Compare Types enum list
		origTypes := original.Types()
		resTypes := restored.Types()
		switch {
		case origTypes.Len() == 0 && resTypes.Len() != 0:
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Types should be empty", test.name)
		case origTypes.Len() != 0 && resTypes.Len() == 0:
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Types should not be empty", test.name)
		case origTypes.Len() != 0 && resTypes.Len() != 0:
			origSlice := origTypes.Slice()
			resSlice := resTypes.Slice()
			if diff := pretty.Compare(origSlice, resSlice); diff != "" {
				t.Errorf("TestUnmarshalRoundTripVehicle(%s): Types mismatch: -want/+got:\n%s", test.name, diff)
			}
		}

		// Compare Bools list
		origBools := original.Bools()
		resBools := restored.Bools()
		switch {
		case origBools.Len() == 0 && resBools.Len() != 0:
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Bools should be empty", test.name)
		case origBools.Len() != 0 && resBools.Len() == 0:
			t.Errorf("TestUnmarshalRoundTripVehicle(%s): Bools should not be empty", test.name)
		case origBools.Len() != 0 && resBools.Len() != 0:
			origSlice := origBools.Slice()
			resSlice := resBools.Slice()
			if diff := pretty.Compare(origSlice, resSlice); diff != "" {
				t.Errorf("TestUnmarshalRoundTripVehicle(%s): Bools mismatch: -want/+got:\n%s", test.name, diff)
			}
		}
	}
}
