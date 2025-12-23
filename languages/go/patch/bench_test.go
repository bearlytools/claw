package patch

import (
	"testing"

	cars "github.com/bearlytools/claw/claw_vendor/github.com/bearlytools/test_claw_imports/cars/claw"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

func BenchmarkDiffNoChanges(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2023).SetModel(cars.GT)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Diff(from, to)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiffOneField(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.GT)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Diff(from, to)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiffTwoFields(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.Venza)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Diff(from, to)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplyOneField(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.GT)
	patch, err := Diff(from, to)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base := cars.NewCar().SetYear(2023).SetModel(cars.GT)
		if err := Apply(base, patch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplyTwoFields(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.Venza)
	patch, err := Diff(from, to)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base := cars.NewCar().SetYear(2023).SetModel(cars.GT)
		if err := Apply(base, patch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPatchMarshal(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.Venza)
	patch, err := Diff(from, to)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := patch.Marshal()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPatchUnmarshal(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.Venza)
	patch, err := Diff(from, to)
	if err != nil {
		b.Fatal(err)
	}
	data, err := patch.Marshal()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := msgs.NewPatch()
		if err := p.Unmarshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundTrip(b *testing.B) {
	from := cars.NewCar().SetYear(2023).SetModel(cars.GT)
	to := cars.NewCar().SetYear(2024).SetModel(cars.Venza)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Diff
		patch, err := Diff(from, to)
		if err != nil {
			b.Fatal(err)
		}

		// Marshal
		data, err := patch.Marshal()
		if err != nil {
			b.Fatal(err)
		}

		// Unmarshal
		p := msgs.NewPatch()
		if err := p.Unmarshal(data); err != nil {
			b.Fatal(err)
		}

		// Apply
		base := cars.NewCar().SetYear(2023).SetModel(cars.GT)
		if err := Apply(base, p); err != nil {
			b.Fatal(err)
		}
	}
}
