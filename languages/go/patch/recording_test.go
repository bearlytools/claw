package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	trucks "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/trucks"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
)

// Alias OpType constants from segment for readability
const (
	opListReplace = segment.OpListReplace
	opListInsert  = segment.OpListInsert
)

func TestRecordingScalarFields(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		modify  func(c cars.Car)
		wantOps int
	}{
		{
			name: "Success: record single field set",
			modify: func(c cars.Car) {
				c.SetYear(2024)
			},
			wantOps: 1,
		},
		{
			name: "Success: record multiple field sets",
			modify: func(c cars.Car) {
				c.SetYear(2024)
				c.SetModel(cars.Venza)
			},
			wantOps: 2,
		},
		{
			name: "Success: record set then clear (zero value)",
			modify: func(c cars.Car) {
				c.SetYear(2024)
				c.SetYear(0) // Clear
			},
			wantOps: 2, // Both set and clear are recorded
		},
		{
			name: "Success: no modifications records nothing",
			modify: func(c cars.Car) {
				// Do nothing
			},
			wantOps: 0,
		},
	}

	for _, test := range tests {
		c := cars.NewCar(ctx)
		c.SetRecording(true)

		test.modify(c)

		ops := c.DrainRecordedOps()
		if len(ops) != test.wantOps {
			t.Errorf("TestRecordingScalarFields(%s): got %d ops, want %d", test.name, len(ops), test.wantOps)
		}
	}
}

func TestRecordingListBools(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		modify  func(v vehicles.Vehicle)
		wantOps int
		wantOp  uint8
	}{
		{
			name: "Success: record SetBools",
			modify: func(v vehicles.Vehicle) {
				v.SetBools(true, false, true)
			},
			wantOps: 1,
			wantOp:  opListReplace,
		},
		{
			name: "Success: record AppendBools",
			modify: func(v vehicles.Vehicle) {
				v.SetBools(true) // First set the list
				v.DrainRecordedOps()
				v.Bools().Append(false) // Then append
			},
			wantOps: 1,
			wantOp:  opListInsert,
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicle(ctx)
		v.SetRecording(true)

		test.modify(v)

		ops := v.DrainRecordedOps()
		if len(ops) != test.wantOps {
			t.Errorf("TestRecordingListBools(%s): got %d ops, want %d", test.name, len(ops), test.wantOps)
			continue
		}

		if test.wantOps > 0 && ops[0].OpType != test.wantOp {
			t.Errorf("TestRecordingListBools(%s): got op type %d, want %d", test.name, ops[0].OpType, test.wantOp)
		}
	}
}

func TestRecordingListNumbers(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		modify  func(v vehicles.Vehicle)
		wantOps int
		wantOp  uint8
	}{
		{
			name: "Success: record SetTypes (enum list)",
			modify: func(v vehicles.Vehicle) {
				v.SetTypes(vehicles.Car, vehicles.Truck)
			},
			wantOps: 1,
			wantOp:  opListReplace,
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicle(ctx)
		v.SetRecording(true)

		test.modify(v)

		ops := v.DrainRecordedOps()
		if len(ops) != test.wantOps {
			t.Errorf("TestRecordingListNumbers(%s): got %d ops, want %d", test.name, len(ops), test.wantOps)
			continue
		}

		if test.wantOps > 0 && ops[0].OpType != test.wantOp {
			t.Errorf("TestRecordingListNumbers(%s): got op type %d, want %d", test.name, ops[0].OpType, test.wantOp)
		}
	}
}

func TestRecordingListStructs(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		modify  func(v vehicles.Vehicle)
		wantOps int
		wantOp  uint8
	}{
		{
			name: "Success: record TruckAppend",
			modify: func(v vehicles.Vehicle) {
				v.TruckAppend(trucks.NewTruck(ctx).SetYear(2023))
			},
			wantOps: 1,
			wantOp:  opListInsert,
		},
		{
			name: "Success: record multiple TruckAppends",
			modify: func(v vehicles.Vehicle) {
				v.TruckAppend(trucks.NewTruck(ctx).SetYear(2023))
				v.TruckAppend(trucks.NewTruck(ctx).SetYear(2024))
			},
			wantOps: 2,
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicle(ctx)
		v.SetRecording(true)

		test.modify(v)

		ops := v.DrainRecordedOps()
		if len(ops) != test.wantOps {
			t.Errorf("TestRecordingListStructs(%s): got %d ops, want %d", test.name, len(ops), test.wantOps)
			continue
		}

		if test.wantOps > 0 && test.wantOp != 0 && ops[0].OpType != test.wantOp {
			t.Errorf("TestRecordingListStructs(%s): got op type %d, want %d", test.name, ops[0].OpType, test.wantOp)
		}
	}
}

func TestRecordingNestedStruct(t *testing.T) {
	ctx := t.Context()
	tests := []struct {
		name    string
		modify  func(v vehicles.Vehicle)
		wantOps int
	}{
		{
			name: "Success: record SetCar",
			modify: func(v vehicles.Vehicle) {
				v.SetCar(cars.NewCar(ctx).SetYear(2024))
			},
			wantOps: 1,
		},
		{
			name: "Success: record clear nested struct (nil)",
			modify: func(v vehicles.Vehicle) {
				v.SetCar(cars.NewCar(ctx).SetYear(2024))
				v.DrainRecordedOps()
				v.SetCar(cars.Car{}) // Clear with zero value
			},
			wantOps: 1,
		},
	}

	for _, test := range tests {
		v := vehicles.NewVehicle(ctx)
		v.SetRecording(true)

		test.modify(v)

		ops := v.DrainRecordedOps()
		if len(ops) != test.wantOps {
			t.Errorf("TestRecordingNestedStruct(%s): got %d ops, want %d", test.name, len(ops), test.wantOps)
		}
	}
}

func TestRecordingDrainClearsOps(t *testing.T) {
	ctx := t.Context()
	c := cars.NewCar(ctx)
	c.SetRecording(true)

	c.SetYear(2024)
	c.SetModel(cars.Venza)

	ops1 := c.DrainRecordedOps()
	if len(ops1) != 2 {
		t.Errorf("TestRecordingDrainClearsOps: first drain got %d ops, want 2", len(ops1))
	}

	ops2 := c.DrainRecordedOps()
	if len(ops2) != 0 {
		t.Errorf("TestRecordingDrainClearsOps: second drain got %d ops, want 0", len(ops2))
	}

	c.SetYear(2025)
	ops3 := c.DrainRecordedOps()
	if len(ops3) != 1 {
		t.Errorf("TestRecordingDrainClearsOps: third drain got %d ops, want 1", len(ops3))
	}
}

func TestRecordingToggle(t *testing.T) {
	ctx := t.Context()
	c := cars.NewCar(ctx)

	if c.Recording() {
		t.Error("TestRecordingToggle: expected Recording() to be false initially")
	}

	c.SetRecording(true)
	if !c.Recording() {
		t.Error("TestRecordingToggle: expected Recording() to be true after enabling")
	}

	c.SetYear(2024)

	c.SetRecording(false)
	if c.Recording() {
		t.Error("TestRecordingToggle: expected Recording() to be false after disabling")
	}

	c.SetYear(2025)

	ops := c.DrainRecordedOps()
	if len(ops) != 1 {
		t.Errorf("TestRecordingToggle: got %d ops, want 1 (only the change while recording was on)", len(ops))
	}
}

func TestRecordingFieldNum(t *testing.T) {
	ctx := t.Context()
	c := cars.NewCar(ctx)
	c.SetRecording(true)

	c.SetYear(2024)

	ops := c.DrainRecordedOps()
	if len(ops) != 1 {
		t.Fatalf("TestRecordingFieldNum: got %d ops, want 1", len(ops))
	}

	// Year field is at FieldNum 2 (Model is 1)
	if ops[0].FieldNum != 2 {
		t.Errorf("TestRecordingFieldNum: got FieldNum %d, want 2 (Year field)", ops[0].FieldNum)
	}
}

func TestRecordedOpsLen(t *testing.T) {
	ctx := t.Context()
	c := cars.NewCar(ctx)
	c.SetRecording(true)

	if c.RecordedOpsLen() != 0 {
		t.Errorf("TestRecordedOpsLen: initial len got %d, want 0", c.RecordedOpsLen())
	}

	c.SetYear(2024)
	if c.RecordedOpsLen() != 1 {
		t.Errorf("TestRecordedOpsLen: after 1 set got %d, want 1", c.RecordedOpsLen())
	}

	c.SetModel(cars.GT)
	if c.RecordedOpsLen() != 2 {
		t.Errorf("TestRecordedOpsLen: after 2 sets got %d, want 2", c.RecordedOpsLen())
	}

	c.DrainRecordedOps()
	if c.RecordedOpsLen() != 0 {
		t.Errorf("TestRecordedOpsLen: after drain got %d, want 0", c.RecordedOpsLen())
	}
}
