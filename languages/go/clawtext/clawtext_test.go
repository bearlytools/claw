package clawtext

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gostdlib/base/context"
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
			want: `Manufacturer: Toyota,
Model: Venza,
Year: 2010,
`,
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
			want: `Manufacturer: 3,
Model: 3,
Year: 2023,
`,
		},
		{
			name: "Success: empty car (zero values)",
			setup: func() cars.Car {
				return cars.NewCar(ctx)
			},
			want: `Manufacturer: Unknown,
Model: ModelUnknown,
Year: 0,
`,
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
			t.Errorf("TestMarshalSimpleCar(%s):\ngot:\n%s\nwant:\n%s", test.name, got.String(), test.want)
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
			want: `Type: Car,
Car: {
    Manufacturer: Toyota,
    Model: Venza,
    Year: 2010,
},
Truck: null,
Types: null,
Bools: null,
`,
		},
		{
			name: "Success: vehicle with nil car",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetType(vehicles.Truck)
			},
			want: `Type: Truck,
Car: null,
Truck: null,
Types: null,
Bools: null,
`,
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
			t.Errorf("TestMarshalNestedStruct(%s):\ngot:\n%s\nwant:\n%s", test.name, got.String(), test.want)
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
			want: `Type: Unknown,
Car: null,
Truck: null,
Types: null,
Bools: [true, false, true],
`,
		},
		{
			name: "Success: vehicle with enum list (strings)",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
			want: `Type: Unknown,
Car: null,
Truck: null,
Types: [Car, Truck],
Bools: null,
`,
		},
		{
			name: "Success: vehicle with enum list (numbers)",
			setup: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx).
					SetTypes(vehicles.Car, vehicles.Truck)
			},
			options: []MarshalOption{WithUseEnumNumbers(true)},
			want: `Type: 0,
Car: null,
Truck: null,
Types: [1, 2],
Bools: null,
`,
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
			t.Errorf("TestMarshalLists(%s):\ngot:\n%s\nwant:\n%s", test.name, got.String(), test.want)
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

	want := `Manufacturer: Ford,
Model: GT,
Year: 2020,
`
	if buf.String() != want {
		t.Errorf("TestMarshalWriter:\ngot:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create a car
	original := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)

	// Marshal to clawtext
	marshaled, err := Marshal(ctx, original)
	if err != nil {
		t.Errorf("TestRoundTrip: marshal error: %s", err)
		return
	}
	defer marshaled.Release(ctx)

	// Unmarshal back to a new car
	restored := cars.NewCar(ctx)
	err = Unmarshal(ctx, marshaled.Bytes(), &restored)
	if err != nil {
		t.Errorf("TestRoundTrip: unmarshal error: %s", err)
		return
	}

	// Compare
	if diff := pretty.Compare(original.Manufacturer(), restored.Manufacturer()); diff != "" {
		t.Errorf("TestRoundTrip: Manufacturer -want/+got:\n%s", diff)
	}
	if diff := pretty.Compare(original.Model(), restored.Model()); diff != "" {
		t.Errorf("TestRoundTrip: Model -want/+got:\n%s", diff)
	}
	if diff := pretty.Compare(original.Year(), restored.Year()); diff != "" {
		t.Errorf("TestRoundTrip: Year -want/+got:\n%s", diff)
	}
}

func TestUnmarshalComments(t *testing.T) {
	ctx := context.Background()

	input := `// This is a comment
Manufacturer: Toyota,
// Another comment
Model: Venza,
/* Multi-line
   comment */
Year: 2010,
`

	car := cars.NewCar(ctx)
	err := Unmarshal(ctx, []byte(input), &car)
	if err != nil {
		t.Errorf("TestUnmarshalComments: got err == %s, want err == nil", err)
		return
	}

	if car.Manufacturer() != manufacturers.Toyota {
		t.Errorf("TestUnmarshalComments: Manufacturer = %v, want Toyota", car.Manufacturer())
	}
	if car.Model() != cars.Venza {
		t.Errorf("TestUnmarshalComments: Model = %v, want Venza", car.Model())
	}
	if car.Year() != 2010 {
		t.Errorf("TestUnmarshalComments: Year = %d, want 2010", car.Year())
	}
}

func TestUnmarshalTrailingComma(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{
			name: "Success: with trailing commas",
			input: `Manufacturer: Toyota,
Model: Venza,
Year: 2010,
`,
		},
		{
			name: "Success: without trailing comma on last field",
			input: `Manufacturer: Toyota,
Model: Venza,
Year: 2010
`,
		},
	}

	for _, test := range tests {
		car := cars.NewCar(ctx)
		err := Unmarshal(ctx, []byte(test.input), &car)
		if err != nil {
			t.Errorf("TestUnmarshalTrailingComma(%s): got err == %s, want err == nil", test.name, err)
			continue
		}

		if car.Year() != 2010 {
			t.Errorf("TestUnmarshalTrailingComma(%s): Year = %d, want 2010", test.name, car.Year())
		}
	}
}

func TestMarshalHexBytes(t *testing.T) {
	ctx := context.Background()

	// Find a struct with bytes field - we'll test the option at least
	car := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)

	// Test with hex bytes option (even if no bytes field)
	got, err := Marshal(ctx, car, WithUseHexBytes(true))
	if err != nil {
		t.Errorf("TestMarshalHexBytes: got err == %s, want err == nil", err)
		return
	}
	defer got.Release(ctx)

	// Should still work, just with hex encoding if there were bytes
	if !strings.Contains(got.String(), "Manufacturer: Toyota") {
		t.Errorf("TestMarshalHexBytes: expected 'Manufacturer: Toyota' in output, got: %s", got.String())
	}
}
