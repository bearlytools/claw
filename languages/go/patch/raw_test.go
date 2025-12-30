package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	trucks "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/trucks"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
	"github.com/kylelemons/godebug/pretty"
)

func TestNewFromRawScalars(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		raw     cars.CarRaw
		wantRaw cars.CarRaw
	}{
		{
			name: "Success: all fields set",
			raw: cars.CarRaw{
				Manufacturer: manufacturers.Toyota,
				Model:        cars.Venza,
				Year:         2024,
			},
			wantRaw: cars.CarRaw{
				Manufacturer: manufacturers.Toyota,
				Model:        cars.Venza,
				Year:         2024,
			},
		},
		{
			name: "Success: partial fields set",
			raw: cars.CarRaw{
				Manufacturer: manufacturers.Toyota,
				Year:         2024,
			},
			wantRaw: cars.CarRaw{
				Manufacturer: manufacturers.Toyota,
				Model:        0,
				Year:         2024,
			},
		},
		{
			name: "Success: empty raw creates empty struct",
			raw:  cars.CarRaw{},
			wantRaw: cars.CarRaw{
				Manufacturer: 0,
				Model:        0,
				Year:         0,
			},
		},
	}

	for _, test := range tests {
		c := cars.NewCarFromRaw(ctx, test.raw)

		got := c.ToRaw()
		if diff := pretty.Compare(test.wantRaw, got); diff != "" {
			t.Errorf("TestNewFromRawScalars(%s): (-want +got)\n%s", test.name, diff)
		}
	}
}

func TestNewFromRawNestedStruct(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		raw     vehicles.VehicleRaw
		wantRaw vehicles.VehicleRaw
	}{
		{
			name: "Success: with nested struct",
			raw: vehicles.VehicleRaw{
				Type: vehicles.Car,
				Car: &cars.CarRaw{
					Manufacturer: manufacturers.Toyota,
					Model:        cars.Venza,
					Year:         2024,
				},
			},
			wantRaw: vehicles.VehicleRaw{
				Type: vehicles.Car,
				Car: &cars.CarRaw{
					Manufacturer: manufacturers.Toyota,
					Model:        cars.Venza,
					Year:         2024,
				},
			},
		},
		{
			name: "Success: without nested struct",
			raw: vehicles.VehicleRaw{
				Type: vehicles.Truck,
			},
			wantRaw: vehicles.VehicleRaw{
				Type: vehicles.Truck,
				Car:  nil,
			},
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicleFromRaw(ctx, test.raw)

		got := v.ToRaw()
		if diff := pretty.Compare(test.wantRaw, got); diff != "" {
			t.Errorf("TestNewFromRawNestedStruct(%s): (-want +got)\n%s", test.name, diff)
		}
	}
}

func TestNewFromRawLists(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		raw     vehicles.VehicleRaw
		wantRaw vehicles.VehicleRaw
	}{
		{
			name: "Success: with bool list",
			raw: vehicles.VehicleRaw{
				Bools: []bool{true, false, true},
			},
			wantRaw: vehicles.VehicleRaw{
				Bools: []bool{true, false, true},
			},
		},
		{
			name: "Success: with enum list",
			raw: vehicles.VehicleRaw{
				Types: []vehicles.Type{vehicles.Car, vehicles.Truck},
			},
			wantRaw: vehicles.VehicleRaw{
				Types: []vehicles.Type{vehicles.Car, vehicles.Truck},
			},
		},
		{
			name: "Success: with struct list",
			raw: vehicles.VehicleRaw{
				Truck: []*trucks.TruckRaw{
					{Year: 2023},
					{Year: 2024},
				},
			},
			wantRaw: vehicles.VehicleRaw{
				Truck: []*trucks.TruckRaw{
					{Year: 2023},
					{Year: 2024},
				},
			},
		},
		{
			name:    "Success: empty lists",
			raw:     vehicles.VehicleRaw{},
			wantRaw: vehicles.VehicleRaw{},
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicleFromRaw(ctx, test.raw)

		got := v.ToRaw()
		if diff := pretty.Compare(test.wantRaw, got); diff != "" {
			t.Errorf("TestNewFromRawLists(%s): (-want +got)\n%s", test.name, diff)
		}
	}
}

func TestToRawRoundtrip(t *testing.T) {
	ctx := t.Context()

	// Create a fully populated vehicle
	v := vehicles.NewVehicle(ctx)
	v.SetType(vehicles.Car)

	c := cars.NewCar(ctx)
	c.SetManufacturer(manufacturers.Toyota)
	c.SetModel(cars.Venza)
	c.SetYear(2024)
	v.SetCar(c)

	v.TruckAppend(trucks.NewTruck(ctx).SetYear(2023))
	v.TruckAppend(trucks.NewTruck(ctx).SetYear(2024))

	v.SetTypes(vehicles.Car, vehicles.Truck)
	v.SetBools(true, false, true)

	// Convert to Raw
	raw := v.ToRaw()

	// Create a new vehicle from Raw
	v2 := vehicles.NewVehicleFromRaw(ctx, raw)

	// Convert back to Raw and compare
	got := v2.ToRaw()

	if diff := pretty.Compare(raw, got); diff != "" {
		t.Errorf("TestToRawRoundtrip: (-want +got)\n%s", diff)
	}
}

func TestAppendRaw(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name      string
		setup     func(v vehicles.Vehicle)
		appendRaw []*trucks.TruckRaw
		wantLen   int
		wantYears []uint16
	}{
		{
			name:  "Success: append to empty list",
			setup: func(v vehicles.Vehicle) {},
			appendRaw: []*trucks.TruckRaw{
				{Year: 2023},
				{Year: 2024},
			},
			wantLen:   2,
			wantYears: []uint16{2023, 2024},
		},
		{
			name: "Success: append to existing list",
			setup: func(v vehicles.Vehicle) {
				v.TruckAppend(trucks.NewTruck(ctx).SetYear(2020))
			},
			appendRaw: []*trucks.TruckRaw{
				{Year: 2023},
			},
			wantLen:   2,
			wantYears: []uint16{2020, 2023},
		},
		{
			name:  "Success: skip nil entries",
			setup: func(v vehicles.Vehicle) {},
			appendRaw: []*trucks.TruckRaw{
				{Year: 2023},
				nil,
				{Year: 2024},
			},
			wantLen:   2,
			wantYears: []uint16{2023, 2024},
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicle(ctx)
		test.setup(v)
		v.TruckAppendRaw(ctx, test.appendRaw...)

		if v.TruckLen() != test.wantLen {
			t.Errorf("TestAppendRaw(%s): got len %d, want %d", test.name, v.TruckLen(), test.wantLen)
			continue
		}

		for i, wantYear := range test.wantYears {
			gotYear := v.TruckGet(i).Year()
			if gotYear != wantYear {
				t.Errorf("TestAppendRaw(%s): index %d got year %d, want %d", test.name, i, gotYear, wantYear)
			}
		}
	}
}

// TestNewFromRawVerifyFields verifies that NewFromRaw correctly sets the underlying
// struct fields, not just testing via ToRaw roundtrip.
func TestNewFromRawVerifyFields(t *testing.T) {
	ctx := t.Context()

	// Test scalar fields
	c := cars.NewCarFromRaw(ctx, cars.CarRaw{
		Manufacturer: manufacturers.Toyota,
		Model:        cars.Venza,
		Year:         2024,
	})

	if c.Manufacturer() != manufacturers.Toyota {
		t.Errorf("TestNewFromRawVerifyFields: Manufacturer got %v, want %v", c.Manufacturer(), manufacturers.Toyota)
	}
	if c.Model() != cars.Venza {
		t.Errorf("TestNewFromRawVerifyFields: Model got %v, want %v", c.Model(), cars.Venza)
	}
	if c.Year() != 2024 {
		t.Errorf("TestNewFromRawVerifyFields: Year got %v, want %v", c.Year(), 2024)
	}

	// Test nested struct
	v := vehicles.NewVehicleFromRaw(ctx, vehicles.VehicleRaw{
		Type: vehicles.Car,
		Car: &cars.CarRaw{
			Manufacturer: manufacturers.Ford,
			Model:        cars.GT,
			Year:         2023,
		},
	})

	if v.Type() != vehicles.Car {
		t.Errorf("TestNewFromRawVerifyFields: Type got %v, want %v", v.Type(), vehicles.Car)
	}
	if v.Car().Manufacturer() != manufacturers.Ford {
		t.Errorf("TestNewFromRawVerifyFields: Car.Manufacturer got %v, want %v", v.Car().Manufacturer(), manufacturers.Ford)
	}
	if v.Car().Model() != cars.GT {
		t.Errorf("TestNewFromRawVerifyFields: Car.Model got %v, want %v", v.Car().Model(), cars.GT)
	}
	if v.Car().Year() != 2023 {
		t.Errorf("TestNewFromRawVerifyFields: Car.Year got %v, want %v", v.Car().Year(), 2023)
	}

	// Test list fields
	v2 := vehicles.NewVehicleFromRaw(ctx, vehicles.VehicleRaw{
		Bools: []bool{true, false, true},
		Types: []vehicles.Type{vehicles.Car, vehicles.Truck},
		Truck: []*trucks.TruckRaw{
			{Year: 2020},
			{Year: 2021},
		},
	})

	bools := v2.Bools().Slice()
	if len(bools) != 3 || bools[0] != true || bools[1] != false || bools[2] != true {
		t.Errorf("TestNewFromRawVerifyFields: Bools got %v, want [true false true]", bools)
	}

	types := v2.Types().Slice()
	if len(types) != 2 || types[0] != vehicles.Car || types[1] != vehicles.Truck {
		t.Errorf("TestNewFromRawVerifyFields: Types got %v, want [Car Truck]", types)
	}

	if v2.TruckLen() != 2 {
		t.Errorf("TestNewFromRawVerifyFields: TruckLen got %d, want 2", v2.TruckLen())
	}
	if v2.TruckGet(0).Year() != 2020 {
		t.Errorf("TestNewFromRawVerifyFields: Truck[0].Year got %d, want 2020", v2.TruckGet(0).Year())
	}
	if v2.TruckGet(1).Year() != 2021 {
		t.Errorf("TestNewFromRawVerifyFields: Truck[1].Year got %d, want 2021", v2.TruckGet(1).Year())
	}
}

// TestNewFromRawMarshalRoundtrip verifies that structs created from Raw can be
// marshaled and unmarshaled correctly.
func TestNewFromRawMarshalRoundtrip(t *testing.T) {
	ctx := t.Context()

	// Create a complex struct from Raw
	original := vehicles.NewVehicleFromRaw(ctx, vehicles.VehicleRaw{
		Type: vehicles.Car,
		Car: &cars.CarRaw{
			Manufacturer: manufacturers.Toyota,
			Model:        cars.ModelS,
			Year:         2024,
		},
		Truck: []*trucks.TruckRaw{
			{Year: 2023},
			{Year: 2024},
		},
		Types: []vehicles.Type{vehicles.Car, vehicles.Truck, vehicles.Car},
		Bools: []bool{true, false, true, false},
	})

	// Marshal to bytes
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("TestNewFromRawMarshalRoundtrip: Marshal error: %v", err)
	}

	// Unmarshal into new struct
	restored := vehicles.NewVehicle(ctx)
	if err := restored.Unmarshal(data); err != nil {
		t.Fatalf("TestNewFromRawMarshalRoundtrip: Unmarshal error: %v", err)
	}

	// Compare using ToRaw
	originalRaw := original.ToRaw()
	restoredRaw := restored.ToRaw()

	if diff := pretty.Compare(originalRaw, restoredRaw); diff != "" {
		t.Errorf("TestNewFromRawMarshalRoundtrip: (-want +got)\n%s", diff)
	}
}

// TestMixRawAndRegularAPI verifies that Raw and regular APIs can be mixed.
func TestMixRawAndRegularAPI(t *testing.T) {
	ctx := t.Context()

	// Start with regular API
	v := vehicles.NewVehicle(ctx)
	v.SetType(vehicles.Truck)

	// Add items using AppendRaw
	v.TruckAppendRaw(ctx, &trucks.TruckRaw{Year: 2020})

	// Add items using regular API
	v.TruckAppend(trucks.NewTruck(ctx).SetYear(2021))

	// Add more using AppendRaw
	v.TruckAppendRaw(ctx, &trucks.TruckRaw{Year: 2022})

	// Verify
	if v.TruckLen() != 3 {
		t.Errorf("TestMixRawAndRegularAPI: TruckLen got %d, want 3", v.TruckLen())
	}

	years := []uint16{2020, 2021, 2022}
	for i, want := range years {
		got := v.TruckGet(i).Year()
		if got != want {
			t.Errorf("TestMixRawAndRegularAPI: Truck[%d].Year got %d, want %d", i, got, want)
		}
	}

	// Set nested struct using NewFromRaw, then use regular setter
	car := cars.NewCarFromRaw(ctx, cars.CarRaw{
		Manufacturer: manufacturers.Toyota,
		Year:         2024,
	})
	v.SetCar(car)

	if v.Car().Manufacturer() != manufacturers.Toyota {
		t.Errorf("TestMixRawAndRegularAPI: Car.Manufacturer got %v, want Toyota", v.Car().Manufacturer())
	}
	if v.Car().Year() != 2024 {
		t.Errorf("TestMixRawAndRegularAPI: Car.Year got %d, want 2024", v.Car().Year())
	}
}

// TestNewFromRawWithNilsInStructList verifies that nil entries in struct lists are skipped.
func TestNewFromRawWithNilsInStructList(t *testing.T) {
	ctx := t.Context()

	v := vehicles.NewVehicleFromRaw(ctx, vehicles.VehicleRaw{
		Truck: []*trucks.TruckRaw{
			{Year: 2020},
			nil,
			{Year: 2022},
			nil,
			{Year: 2024},
		},
	})

	// Should only have 3 trucks (nils skipped)
	if v.TruckLen() != 3 {
		t.Errorf("TestNewFromRawWithNilsInStructList: TruckLen got %d, want 3", v.TruckLen())
	}

	expectedYears := []uint16{2020, 2022, 2024}
	for i, want := range expectedYears {
		got := v.TruckGet(i).Year()
		if got != want {
			t.Errorf("TestNewFromRawWithNilsInStructList: Truck[%d].Year got %d, want %d", i, got, want)
		}
	}
}

// TestToRawPreservesZeroValues verifies ToRaw correctly handles zero vs unset.
func TestToRawPreservesZeroValues(t *testing.T) {
	ctx := t.Context()

	// Create struct with only some fields set
	c := cars.NewCar(ctx)
	c.SetYear(2024)
	// Manufacturer and Model are not set (zero value)

	raw := c.ToRaw()

	if raw.Year != 2024 {
		t.Errorf("TestToRawPreservesZeroValues: Year got %d, want 2024", raw.Year)
	}
	if raw.Manufacturer != 0 {
		t.Errorf("TestToRawPreservesZeroValues: Manufacturer got %v, want 0", raw.Manufacturer)
	}
	if raw.Model != 0 {
		t.Errorf("TestToRawPreservesZeroValues: Model got %v, want 0", raw.Model)
	}
}

// TestNewFromRawEmptyStruct verifies creating from an empty Raw struct.
func TestNewFromRawEmptyStruct(t *testing.T) {
	ctx := t.Context()

	v := vehicles.NewVehicleFromRaw(ctx, vehicles.VehicleRaw{})

	if v.Type() != 0 {
		t.Errorf("TestNewFromRawEmptyStruct: Type got %v, want 0", v.Type())
	}
	if v.TruckLen() != 0 {
		t.Errorf("TestNewFromRawEmptyStruct: TruckLen got %d, want 0", v.TruckLen())
	}

	// ToRaw should return empty struct
	raw := v.ToRaw()
	if raw.Type != 0 || raw.Car != nil || raw.Truck != nil || raw.Types != nil || raw.Bools != nil {
		t.Errorf("TestNewFromRawEmptyStruct: ToRaw should return empty struct, got %+v", raw)
	}
}
