package clawtext

import (
	"bytes"
	"context"
	"strings"
	"testing"

	vehicles "github.com/bearlytools/claw/testing/imports/vehicles/claw"
	"github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers"
	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
)

// createSimpleCar creates a simple car for benchmarking.
func createSimpleCar(ctx context.Context) cars.Car {
	return cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)
}

// createVehicleWithCar creates a vehicle with a nested car for benchmarking.
func createVehicleWithCar(ctx context.Context) vehicles.Vehicle {
	car := cars.NewCar(ctx).
		SetManufacturer(manufacturers.Toyota).
		SetModel(cars.Venza).
		SetYear(2010)
	return vehicles.NewVehicle(ctx).
		SetType(vehicles.Car).
		SetCar(car)
}

// createVehicleWithLists creates a vehicle with lists for benchmarking.
func createVehicleWithLists(ctx context.Context) vehicles.Vehicle {
	return vehicles.NewVehicle(ctx).
		SetBools(true, false, true, false, true).
		SetTypes(vehicles.Car, vehicles.Truck, vehicles.Car)
}

// createFullCar creates a car with all fields populated for benchmarking.
func createFullCar(ctx context.Context) cars.Car {
	return cars.NewCar(ctx).
		SetManufacturer(manufacturers.Ford).
		SetModel(cars.GT).
		SetYear(2020)
}

// Marshaling Benchmarks

func BenchmarkMarshalSimpleCar(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalNestedStruct(b *testing.B) {
	ctx := context.Background()
	vehicle := createVehicleWithCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, vehicle)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalWithLists(b *testing.B) {
	ctx := context.Background()
	vehicle := createVehicleWithLists(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, vehicle)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalEnumAsNumbers(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car, WithUseEnumNumbers(true))
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalHexBytes(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car, WithUseHexBytes(true))
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkMarshalWriter(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := MarshalWriter(ctx, car, &buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalFullCar(b *testing.B) {
	ctx := context.Background()
	car := createFullCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

// Unmarshaling Benchmarks

func BenchmarkUnmarshalSimpleCar(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	textData := make([]byte, buf.Len())
	copy(textData, buf.Bytes())
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := cars.NewCar(ctx)
		err := Unmarshal(ctx, textData, &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalFullCar(b *testing.B) {
	ctx := context.Background()
	car := createFullCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	textData := make([]byte, buf.Len())
	copy(textData, buf.Bytes())
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := cars.NewCar(ctx)
		err := Unmarshal(ctx, textData, &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalReader(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	textStr := buf.String()
	buf.Release(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		restored := cars.NewCar(ctx)
		err := UnmarshalReader(ctx, strings.NewReader(textStr), &restored)
		if err != nil {
			b.Fatal(err)
		}
	}
}


// Round-trip Benchmarks

func BenchmarkRoundTripSimple(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		restored := cars.NewCar(ctx)
		err = Unmarshal(ctx, buf.Bytes(), &restored)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

func BenchmarkRoundTripFullCar(b *testing.B) {
	ctx := context.Background()
	car := createFullCar(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf, err := Marshal(ctx, car)
		if err != nil {
			b.Fatal(err)
		}
		restored := cars.NewCar(ctx)
		err = Unmarshal(ctx, buf.Bytes(), &restored)
		if err != nil {
			b.Fatal(err)
		}
		buf.Release(ctx)
	}
}

// Comparison benchmarks between bytes vs writer

func BenchmarkMarshalBytesVsWriter(b *testing.B) {
	ctx := context.Background()
	car := createFullCar(ctx)

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf, err := Marshal(ctx, car)
			if err != nil {
				b.Fatal(err)
			}
			buf.Release(ctx)
		}
	})

	b.Run("MarshalWriter", func(b *testing.B) {
		var buf bytes.Buffer
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			err := MarshalWriter(ctx, car, &buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUnmarshalBytesVsReader(b *testing.B) {
	ctx := context.Background()
	car := createSimpleCar(ctx)
	buf, err := Marshal(ctx, car)
	if err != nil {
		b.Fatal(err)
	}
	textData := make([]byte, buf.Len())
	copy(textData, buf.Bytes())
	textStr := buf.String()
	buf.Release(ctx)

	b.Run("Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			restored := cars.NewCar(ctx)
			err := Unmarshal(ctx, textData, &restored)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("UnmarshalReader", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			restored := cars.NewCar(ctx)
			err := UnmarshalReader(ctx, strings.NewReader(textStr), &restored)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
