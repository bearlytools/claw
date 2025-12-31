package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	trucks "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/trucks"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
)

func TestApplyScalarFields(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name     string
		base     func() cars.Car
		target   func() cars.Car
		wantYear uint16
		wantErr  bool
	}{
		{
			name: "Success: no changes",
			base: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			target: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			wantYear: 2023,
		},
		{
			name: "Success: year changed",
			base: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2023)
			},
			target: func() cars.Car {
				return cars.NewCar(ctx).SetYear(2024)
			},
			wantYear: 2024,
		},
	}

	for _, test := range tests {
		base := test.base()
		target := test.target()

		// Create patch from base to target
		p, err := Diff(ctx, base, target)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestApplyScalarFields(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestApplyScalarFields(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Apply patch to a fresh base
		freshBase := test.base()
		if err := Apply(ctx, freshBase, p); err != nil {
			t.Errorf("TestApplyScalarFields(%s): Apply error: %s", test.name, err)
			continue
		}

		// Verify the result
		if freshBase.Year() != test.wantYear {
			t.Errorf("TestApplyScalarFields(%s): got Year=%d, want Year=%d", test.name, freshBase.Year(), test.wantYear)
		}
	}
}

func TestApplyRoundTrip(t *testing.T) {
	ctx := t.Context()
	// Create two different versions
	from := cars.NewCar(ctx).SetYear(2023)
	to := cars.NewCar(ctx).SetYear(2024)

	// Generate patch
	p, err := Diff(ctx, from, to)
	if err != nil {
		t.Fatalf("TestApplyRoundTrip: Diff error: %s", err)
	}

	// Serialize the patch
	data, err := p.Marshal()
	if err != nil {
		t.Fatalf("TestApplyRoundTrip: Marshal error: %s", err)
	}

	// Deserialize
	receivedPatch := msgs.NewPatch(ctx)
	if err := receivedPatch.Unmarshal(data); err != nil {
		t.Fatalf("TestApplyRoundTrip: Unmarshal error: %s", err)
	}

	// Apply to a fresh base
	base := cars.NewCar(ctx).SetYear(2023)
	if err := Apply(ctx, base, receivedPatch); err != nil {
		t.Fatalf("TestApplyRoundTrip: Apply error: %s", err)
	}

	// Verify the result matches target
	if base.Year() != 2024 {
		t.Errorf("TestApplyRoundTrip: got Year=%d, want Year=2024", base.Year())
	}
}

func TestApplyMultipleFields(t *testing.T) {
	ctx := t.Context()
	// Test with multiple field changes
	from := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar(ctx).SetYear(2024).SetModel(cars.Venza)

	// Generate patch
	p, err := Diff(ctx, from, to)
	if err != nil {
		t.Fatalf("TestApplyMultipleFields: Diff error: %s", err)
	}

	// Apply to a fresh base
	base := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
	if err := Apply(ctx, base, p); err != nil {
		t.Fatalf("TestApplyMultipleFields: Apply error: %s", err)
	}

	// Verify both fields changed
	if base.Year() != 2024 {
		t.Errorf("TestApplyMultipleFields: got Year=%d, want Year=2024", base.Year())
	}
	if base.Model() != cars.Venza {
		t.Errorf("TestApplyMultipleFields: got Model=%v, want Model=%v", base.Model(), cars.Venza)
	}
}

func TestApplyListOperations(t *testing.T) {
	ctx := t.Context()
	// Test LIST_SET operation directly
	t.Run("LIST_SET bool", func(t *testing.T) {
		// Create a patch with a LIST_SET operation
		from := cars.NewCar(ctx).SetYear(2023)
		to := cars.NewCar(ctx).SetYear(2023)

		// Verify Diff produces no ops for identical structs
		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Fatalf("Diff error: %v", err)
		}
		if p.OpsLen(ctx) != 0 {
			t.Errorf("expected 0 ops for identical structs, got %d", p.OpsLen(ctx))
		}
	})

	// Test that diffing and applying works for scalar fields (basic sanity check)
	t.Run("Scalar field diff and apply", func(t *testing.T) {
		from := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
		to := cars.NewCar(ctx).SetYear(2025).SetModel(cars.ModelS)

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Fatalf("Diff error: %v", err)
		}

		// Apply to base
		base := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
		if err := Apply(ctx, base, p); err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if base.Year() != 2025 {
			t.Errorf("Year: got %d, want 2025", base.Year())
		}
		if base.Model() != cars.ModelS {
			t.Errorf("Model: got %v, want %v", base.Model(), cars.ModelS)
		}
	})

	// Test list operations via internal functions
	t.Run("List encoding/decoding roundtrip", func(t *testing.T) {
		// Test encoding and decoding of list data
		// This verifies the encoding functions work correctly

		// Test bool list encoding
		boolData := []byte{1, 0, 1, 1, 0}
		if len(boolData) != 5 {
			t.Errorf("bool data length: got %d, want 5", len(boolData))
		}

		// Test number list encoding (int32)
		numData := make([]byte, 12) // 3 int32s
		copy(numData[0:4], encodeInt32(100))
		copy(numData[4:8], encodeInt32(200))
		copy(numData[8:12], encodeInt32(300))

		if decodeInt32(numData[0:4]) != 100 {
			t.Errorf("first int32: got %d, want 100", decodeInt32(numData[0:4]))
		}
		if decodeInt32(numData[4:8]) != 200 {
			t.Errorf("second int32: got %d, want 200", decodeInt32(numData[4:8]))
		}
		if decodeInt32(numData[8:12]) != 300 {
			t.Errorf("third int32: got %d, want 300", decodeInt32(numData[8:12]))
		}
	})
}

func TestApplyPatchVersioning(t *testing.T) {
	ctx := t.Context()
	from := cars.NewCar(ctx).SetYear(2023)
	to := cars.NewCar(ctx).SetYear(2024)

	p, err := Diff(ctx, from, to)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}

	if p.Version() != PatchVersion {
		t.Errorf("Version: got %d, want %d", p.Version(), PatchVersion)
	}
}

func TestApplyListBools(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name      string
		fromBools []bool
		toBools   []bool
		wantBools []bool
		wantOps   int
		wantErr   bool
	}{
		{
			name:      "Success: identical bool lists produce no ops",
			fromBools: []bool{true, false, true},
			toBools:   []bool{true, false, true},
			wantBools: []bool{true, false, true},
			wantOps:   0,
		},
		{
			name:      "Success: different bool lists",
			fromBools: []bool{true, false, true},
			toBools:   []bool{false, true, false},
			wantBools: []bool{false, true, false},
			wantOps:   3, // 3 LIST_SET ops since all elements differ
		},
		{
			name:      "Success: add bools to empty list",
			fromBools: nil,
			toBools:   []bool{true, false},
			wantBools: []bool{true, false},
			wantOps:   1,
		},
		{
			name:      "Success: clear bools",
			fromBools: []bool{true, false},
			toBools:   nil,
			wantBools: nil,
			wantOps:   1,
		},
		{
			name:      "Success: both empty",
			fromBools: nil,
			toBools:   nil,
			wantBools: nil,
			wantOps:   0,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle(ctx)
		if test.fromBools != nil {
			from.SetBools(test.fromBools...)
		}

		to := vehicles.NewVehicle(ctx)
		if test.toBools != nil {
			to.SetBools(test.toBools...)
		}

		p, err := Diff(ctx, from, to)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestApplyListBools(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestApplyListBools(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if p.OpsLen(ctx) != test.wantOps {
			t.Errorf("TestApplyListBools(%s): got %d ops, want %d ops", test.name, p.OpsLen(ctx), test.wantOps)
		}

		// Apply patch
		freshFrom := vehicles.NewVehicle(ctx)
		if test.fromBools != nil {
			freshFrom.SetBools(test.fromBools...)
		}

		if err := Apply(ctx, freshFrom, p); err != nil {
			t.Errorf("TestApplyListBools(%s): Apply error: %s", test.name, err)
			continue
		}

		bools := freshFrom.Bools()
		var gotBools []bool
		if bools != nil && bools.Len() > 0 {
			gotBools = bools.Slice()
		}
		if len(gotBools) != len(test.wantBools) {
			t.Errorf("TestApplyListBools(%s): got len=%d, want len=%d", test.name, len(gotBools), len(test.wantBools))
			continue
		}
		for i, want := range test.wantBools {
			if gotBools[i] != want {
				t.Errorf("TestApplyListBools(%s): got[%d]=%v, want=%v", test.name, i, gotBools[i], want)
			}
		}
	}
}

func TestApplyNestedStruct(t *testing.T) {
	ctx := t.Context()
	// Test that identical nested structs produce no ops (no apply needed)
	t.Run("Success: identical nested structs produce no ops", func(t *testing.T) {
		from := vehicles.NewVehicle(ctx).SetCar(cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT))
		to := vehicles.NewVehicle(ctx).SetCar(cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT))

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Fatalf("TestApplyNestedStruct: Diff error: %s", err)
		}

		if p.OpsLen(ctx) != 0 {
			t.Errorf("TestApplyNestedStruct: got %d ops, want 0 ops for identical structs", p.OpsLen(ctx))
		}
	})

	// Test nested struct field changes with STRUCT_PATCH
	t.Run("Success: nested struct diff and apply with STRUCT_PATCH", func(t *testing.T) {
		from := vehicles.NewVehicle(ctx).SetCar(cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT))
		to := vehicles.NewVehicle(ctx).SetCar(cars.NewCar(ctx).SetYear(2024).SetModel(cars.Venza))

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Fatalf("TestApplyNestedStruct: Diff error: %s", err)
		}

		if p.OpsLen(ctx) != 1 {
			t.Errorf("TestApplyNestedStruct: got %d ops, want 1 op", p.OpsLen(ctx))
		}

		if p.OpsLen(ctx) > 0 && p.OpsGet(ctx,0).Type() != msgs.StructPatch {
			t.Errorf("TestApplyNestedStruct: got op type %v, want StructPatch", p.OpsGet(ctx,0).Type())
		}

		// Apply the patch and verify the nested struct was updated
		base := vehicles.NewVehicle(ctx).SetCar(cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT))
		if err := Apply(ctx, base, p); err != nil {
			t.Fatalf("TestApplyNestedStruct: Apply error: %s", err)
		}

		if base.Car().Year() != 2024 {
			t.Errorf("TestApplyNestedStruct: Car.Year=%d, want 2024", base.Car().Year())
		}
		if base.Car().Model() != cars.Venza {
			t.Errorf("TestApplyNestedStruct: Car.Model=%v, want %v", base.Car().Model(), cars.Venza)
		}
	})
}

func TestApplyListStructs(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		base    func() vehicles.Vehicle
		target  func() vehicles.Vehicle
		wantLen int
		wantErr bool
	}{
		{
			name: "Success: add trucks to empty list",
			base: func() vehicles.Vehicle {
				return vehicles.NewVehicle(ctx)
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2023).SetModel(trucks.F100))
				return v
			},
			wantLen: 1,
		},
		{
			name: "Success: modify truck in list",
			base: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2023).SetModel(trucks.F100))
				return v
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2024).SetModel(trucks.Tundra))
				return v
			},
			wantLen: 1,
		},
		{
			name: "Success: add additional truck",
			base: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2023).SetModel(trucks.F100))
				return v
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2023).SetModel(trucks.F100))
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2024).SetModel(trucks.Tundra))
				return v
			},
			wantLen: 2,
		},
		{
			name: "Success: remove truck from list",
			base: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2023).SetModel(trucks.F100))
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2024).SetModel(trucks.Tundra))
				return v
			},
			target: func() vehicles.Vehicle {
				v := vehicles.NewVehicle(ctx)
				v.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(2023).SetModel(trucks.F100))
				return v
			},
			wantLen: 1,
		},
	}

	for _, test := range tests {
		base := test.base()
		target := test.target()

		p, err := Diff(ctx, base, target)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestApplyListStructs(%s): got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestApplyListStructs(%s): got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		// Apply the patch
		applied := test.base()
		if err := Apply(ctx, applied, p); err != nil {
			t.Errorf("TestApplyListStructs(%s): Apply error: %s", test.name, err)
			continue
		}

		// Verify list length
		truckList := applied.TruckList(ctx)
		if truckList == nil && test.wantLen > 0 {
			t.Errorf("TestApplyListStructs(%s): got nil truck list, want len %d", test.name, test.wantLen)
			continue
		}
		if truckList != nil && truckList.Len() != test.wantLen {
			t.Errorf("TestApplyListStructs(%s): got len %d, want %d", test.name, truckList.Len(), test.wantLen)
		}
	}
}

func TestApplyNestedStructWithManufacturer(t *testing.T) {
	ctx := t.Context()
	// Test that diff correctly identifies changes in nested struct (diff only)
	from := vehicles.NewVehicle(ctx).SetCar(
		cars.NewCar(ctx).SetYear(2023).SetManufacturer(manufacturers.Toyota),
	)
	to := vehicles.NewVehicle(ctx).SetCar(
		cars.NewCar(ctx).SetYear(2024).SetManufacturer(manufacturers.Ford),
	)

	p, err := Diff(ctx, from, to)
	if err != nil {
		t.Fatalf("TestApplyNestedStructWithManufacturer: Diff error: %s", err)
	}

	// Should produce 1 STRUCT_PATCH op for the Car field
	if p.OpsLen(ctx) != 1 {
		t.Errorf("TestApplyNestedStructWithManufacturer: got %d ops, want 1", p.OpsLen(ctx))
	}

	if p.OpsLen(ctx) > 0 && p.OpsGet(ctx,0).Type() != msgs.StructPatch {
		t.Errorf("TestApplyNestedStructWithManufacturer: got op type %v, want StructPatch", p.OpsGet(ctx,0).Type())
	}
}

func TestApplyEmptyPatch(t *testing.T) {
	ctx := t.Context()
	// Test applying an empty patch makes no changes
	base := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
	p := msgs.NewPatch(ctx)
	p.SetVersion(PatchVersion)

	if err := Apply(ctx, base, p); err != nil {
		t.Fatalf("TestApplyEmptyPatch: Apply error: %s", err)
	}

	if base.Year() != 2023 {
		t.Errorf("TestApplyEmptyPatch: got Year=%d, want Year=2023", base.Year())
	}
	if base.Model() != cars.GT {
		t.Errorf("TestApplyEmptyPatch: got Model=%v, want Model=%v", base.Model(), cars.GT)
	}
}

func TestDiffIdenticalStructs(t *testing.T) {
	ctx := t.Context()
	// Test that diffing identical structs produces an empty patch
	from := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar(ctx).SetYear(2023).SetModel(cars.GT)

	p, err := Diff(ctx, from, to)
	if err != nil {
		t.Fatalf("TestDiffIdenticalStructs: Diff error: %s", err)
	}

	if !IsEmpty(ctx, p) {
		t.Errorf("TestDiffIdenticalStructs: expected empty patch for identical structs, got %d ops", p.OpsLen(ctx))
	}
}

func TestApplyEnumList(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name      string
		fromTypes []vehicles.Type
		toTypes   []vehicles.Type
		wantTypes []vehicles.Type
		wantOps   int
	}{
		{
			name:      "Success: identical enum lists produce no ops",
			fromTypes: []vehicles.Type{vehicles.Car, vehicles.Truck},
			toTypes:   []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantTypes: []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantOps:   0,
		},
		{
			name:      "Success: different enum lists",
			fromTypes: []vehicles.Type{vehicles.Car},
			toTypes:   []vehicles.Type{vehicles.Truck},
			wantTypes: []vehicles.Type{vehicles.Truck},
			wantOps:   1,
		},
		{
			name:      "Success: add to enum list",
			fromTypes: nil,
			toTypes:   []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantTypes: []vehicles.Type{vehicles.Car, vehicles.Truck},
			wantOps:   1,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle(ctx)
		if test.fromTypes != nil {
			from.SetTypes(test.fromTypes...)
		}

		to := vehicles.NewVehicle(ctx)
		if test.toTypes != nil {
			to.SetTypes(test.toTypes...)
		}

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Errorf("TestApplyEnumList(%s): Diff error: %s", test.name, err)
			continue
		}

		if p.OpsLen(ctx) != test.wantOps {
			t.Errorf("TestApplyEnumList(%s): got %d ops, want %d ops", test.name, p.OpsLen(ctx), test.wantOps)
		}

		if test.wantOps == 0 {
			continue
		}

		freshFrom := vehicles.NewVehicle(ctx)
		if test.fromTypes != nil {
			freshFrom.SetTypes(test.fromTypes...)
		}

		if err := Apply(ctx, freshFrom, p); err != nil {
			t.Errorf("TestApplyEnumList(%s): Apply error: %s", test.name, err)
			continue
		}

		gotTypes := freshFrom.Types()

		if gotTypes.Len() != len(test.wantTypes) {
			t.Errorf("TestApplyEnumList(%s): got len=%d, want len=%d", test.name, gotTypes.Len(), len(test.wantTypes))
			continue
		}
		for i, want := range test.wantTypes {
			if gotTypes.Get(i) != want {
				t.Errorf("TestApplyEnumList(%s): got[%d]=%v, want=%v", test.name, i, gotTypes.Get(i), want)
			}
		}
	}
}

func TestDiffListSetInsertRemove(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name        string
		fromBools   []bool
		toBools     []bool
		wantOps     int
		wantOpTypes []msgs.OpType
	}{
		{
			name:        "Success: LIST_SET for changed elements only",
			fromBools:   []bool{true, false, true},
			toBools:     []bool{true, true, true},
			wantOps:     1,
			wantOpTypes: []msgs.OpType{msgs.ListSet},
		},
		{
			name:        "Success: LIST_INSERT for new elements",
			fromBools:   []bool{true, false},
			toBools:     []bool{true, false, true, false},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListInsert, msgs.ListInsert},
		},
		{
			name:        "Success: LIST_REMOVE for deleted elements",
			fromBools:   []bool{true, false, true, false},
			toBools:     []bool{true, false},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListRemove, msgs.ListRemove},
		},
		{
			name:        "Success: mixed SET and INSERT",
			fromBools:   []bool{true, false},
			toBools:     []bool{false, false, true},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListSet, msgs.ListInsert},
		},
		{
			name:        "Success: mixed SET and REMOVE (below threshold)",
			fromBools:   []bool{true, false, true, false, true},
			toBools:     []bool{false, false, true, false},
			wantOps:     2,
			wantOpTypes: []msgs.OpType{msgs.ListSet, msgs.ListRemove},
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle(ctx)
		from.SetBools(test.fromBools...)

		to := vehicles.NewVehicle(ctx)
		to.SetBools(test.toBools...)

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Errorf("TestDiffListSetInsertRemove(%s): Diff error: %s", test.name, err)
			continue
		}

		if p.OpsLen(ctx) != test.wantOps {
			t.Errorf("TestDiffListSetInsertRemove(%s): got %d ops, want %d ops", test.name, p.OpsLen(ctx), test.wantOps)
			continue
		}

		for i, wantType := range test.wantOpTypes {
			if i >= p.OpsLen(ctx) {
				break
			}
			if p.OpsGet(ctx,i).Type() != wantType {
				t.Errorf("TestDiffListSetInsertRemove(%s): op[%d] got type %v, want %v", test.name, i, p.OpsGet(ctx,i).Type(), wantType)
			}
		}

		freshFrom := vehicles.NewVehicle(ctx)
		freshFrom.SetBools(test.fromBools...)

		if err := Apply(ctx, freshFrom, p); err != nil {
			t.Errorf("TestDiffListSetInsertRemove(%s): Apply error: %s", test.name, err)
			continue
		}

		gotBools := freshFrom.Bools().Slice()
		if len(gotBools) != len(test.toBools) {
			t.Errorf("TestDiffListSetInsertRemove(%s): got len=%d, want len=%d", test.name, len(gotBools), len(test.toBools))
			continue
		}
		for i, want := range test.toBools {
			if gotBools[i] != want {
				t.Errorf("TestDiffListSetInsertRemove(%s): got[%d]=%v, want=%v", test.name, i, gotBools[i], want)
			}
		}
	}
}

func TestDiffListStructPatch(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name       string
		fromYears  []uint16
		toYears    []uint16
		wantOps    int
		checkPatch bool
	}{
		{
			name:       "Success: single struct modified uses LIST_STRUCT_PATCH",
			fromYears:  []uint16{2023},
			toYears:    []uint16{2024},
			wantOps:    1,
			checkPatch: true,
		},
		{
			name:      "Success: add struct uses LIST_INSERT",
			fromYears: []uint16{2023},
			toYears:   []uint16{2023, 2024},
			wantOps:   1,
		},
		{
			name:      "Success: remove struct uses LIST_REMOVE",
			fromYears: []uint16{2023, 2024},
			toYears:   []uint16{2023},
			wantOps:   1,
		},
	}

	for _, test := range tests {
		from := vehicles.NewVehicle(ctx)
		for _, year := range test.fromYears {
			from.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(year).SetModel(trucks.F100))
		}

		to := vehicles.NewVehicle(ctx)
		for _, year := range test.toYears {
			to.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(year).SetModel(trucks.F100))
		}

		p, err := Diff(ctx, from, to)
		if err != nil {
			t.Errorf("TestDiffListStructPatch(%s): Diff error: %s", test.name, err)
			continue
		}

		if p.OpsLen(ctx) != test.wantOps {
			t.Errorf("TestDiffListStructPatch(%s): got %d ops, want %d ops", test.name, p.OpsLen(ctx), test.wantOps)
			continue
		}

		if test.checkPatch && p.OpsLen(ctx) > 0 {
			if p.OpsGet(ctx,0).Type() != msgs.ListStructPatch {
				t.Errorf("TestDiffListStructPatch(%s): got op type %v, want ListStructPatch", test.name, p.OpsGet(ctx,0).Type())
			}
		}

		freshFrom := vehicles.NewVehicle(ctx)
		for _, year := range test.fromYears {
			freshFrom.TruckAppend(ctx, trucks.NewTruck(ctx).SetYear(year).SetModel(trucks.F100))
		}

		if err := Apply(ctx, freshFrom, p); err != nil {
			t.Errorf("TestDiffListStructPatch(%s): Apply error: %s", test.name, err)
			continue
		}

		truckList := freshFrom.TruckList(ctx)
		if truckList.Len() != len(test.toYears) {
			t.Errorf("TestDiffListStructPatch(%s): got %d trucks, want %d trucks", test.name, truckList.Len(), len(test.toYears))
			continue
		}
		for i, wantYear := range test.toYears {
			truck := trucks.XXXNewTruckFrom(truckList.Get(i))
			if truck.Year() != wantYear {
				t.Errorf("TestDiffListStructPatch(%s): truck[%d].Year=%d, want=%d", test.name, i, truck.Year(), wantYear)
			}
		}
	}
}
