package pack

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/gostdlib/base/context"
	"github.com/kylelemons/godebug/pretty"
)

func TestPackUnpack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "Success: empty input",
			input:   []byte{},
			wantErr: false,
		},
		{
			name:    "Success: single zero word",
			input:   make([]byte, 8),
			wantErr: false,
		},
		{
			name:    "Success: single word with one non-zero byte",
			input:   []byte{0x42, 0, 0, 0, 0, 0, 0, 0},
			wantErr: false,
		},
		{
			name:    "Success: single word all non-zero",
			input:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantErr: false,
		},
		{
			name:    "Success: multiple zero words",
			input:   make([]byte, 64),
			wantErr: false,
		},
		{
			name: "Success: typical Claw header pattern",
			input: []byte{
				// Field 1, bool type, value true
				0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
				// Field 2, int32 type, value 42
				0x02, 0x00, 0x04, 0x2a, 0x00, 0x00, 0x00, 0x00,
			},
			wantErr: false,
		},
		{
			name: "Success: mixed zeros and data",
			input: func() []byte {
				b := make([]byte, 80)
				b[0] = 0x42
				b[16] = 0xFF
				b[17] = 0xFF
				b[32] = 0x01
				return b
			}(),
			wantErr: false,
		},
		{
			name: "Success: all 0xFF bytes (worst case)",
			input: func() []byte {
				b := make([]byte, 64)
				for i := range b {
					b[i] = 0xFF
				}
				return b
			}(),
			wantErr: false,
		},
		{
			name:    "Error: input not 8-byte aligned",
			input:   []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()

		packed, err := Pack(ctx, test.input)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestPackUnpack(%s)]: got err == nil, want err != nil", test.name)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("[TestPackUnpack(%s)]: got err == %s, want err == nil", test.name, err)
			continue
		case err != nil:
			continue
		}

		if packed == nil && len(test.input) == 0 {
			continue
		}

		defer packed.Release(ctx)

		unpacked, err := Unpack(ctx, packed.Bytes())
		if err != nil {
			t.Errorf("[TestPackUnpack(%s)]: Unpack failed: %s", test.name, err)
			continue
		}
		defer unpacked.Release(ctx)

		if diff := pretty.Compare(test.input, unpacked.Bytes()); diff != "" {
			t.Errorf("[TestPackUnpack(%s)]: roundtrip mismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestPackCompression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          []byte
		maxRatio       float64 // packed size should be <= this ratio of original
		expectedSaving bool    // true if we expect compression
	}{
		{
			name: "Claw headers compress well",
			input: func() []byte {
				// Simulate 10 Claw field headers with small values
				b := make([]byte, 80)
				for i := 0; i < 10; i++ {
					offset := i * 8
					b[offset] = byte(i + 1)   // field number
					b[offset+2] = byte(i % 8) // field type
					b[offset+3] = byte(i * 2) // small value
				}
				return b
			}(),
			maxRatio:       0.70, // 16-byte header overhead affects small messages
			expectedSaving: true,
		},
		{
			name:           "All zeros compress extremely well",
			input:          make([]byte, 2048),
			maxRatio:       0.02,
			expectedSaving: true,
		},
		{
			name: "Dense data has minimal overhead",
			input: func() []byte {
				b := make([]byte, 64)
				for i := range b {
					b[i] = byte(i + 1) // no zeros
				}
				return b
			}(),
			maxRatio:       1.3, // allow up to 30% overhead for dense data
			expectedSaving: false,
		},
	}

	for _, test := range tests {
		ctx := t.Context()

		packed, err := Pack(ctx, test.input)
		if err != nil {
			t.Errorf("[TestPackCompression(%s)]: Pack failed: %s", test.name, err)
			continue
		}
		defer packed.Release(ctx)

		ratio := float64(packed.Len()) / float64(len(test.input))
		if ratio > test.maxRatio {
			t.Errorf("[TestPackCompression(%s)]: compression ratio %.2f exceeds max %.2f (packed=%d, original=%d)",
				test.name, ratio, test.maxRatio, packed.Len(), len(test.input))
		}

		if test.expectedSaving && packed.Len() >= len(test.input) {
			t.Errorf("[TestPackCompression(%s)]: expected compression but packed (%d) >= original (%d)",
				test.name, packed.Len(), len(test.input))
		}
	}
}

func TestHeaderFunctions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	input := make([]byte, 256)
	input[0] = 0x42

	packed, err := Pack(ctx, input)
	if err != nil {
		t.Fatalf("[TestHeaderFunctions]: Pack failed: %s", err)
	}
	defer packed.Release(ctx)

	if got := UnpackedSize(packed.Bytes()); got != 256 {
		t.Errorf("[TestHeaderFunctions]: UnpackedSize = %d, want 256", got)
	}

	packedDataSize := PackedSize(packed.Bytes())
	if packedDataSize <= 0 {
		t.Errorf("[TestHeaderFunctions]: PackedSize = %d, want > 0", packedDataSize)
	}

	if got := packed.Len(); got != HeaderSize+packedDataSize {
		t.Errorf("[TestHeaderFunctions]: total length = %d, want %d", got, HeaderSize+packedDataSize)
	}

	ratio := CompressionRatio(packed.Bytes())
	if ratio <= 0 || ratio >= 1 {
		t.Errorf("[TestHeaderFunctions]: CompressionRatio = %f, want 0 < ratio < 1", ratio)
	}
}

func TestUnpackErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		packed  []byte
		wantErr bool
	}{
		{
			name:    "Error: too short for header",
			packed:  []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name: "Error: declared packed size larger than data",
			packed: func() []byte {
				b := make([]byte, HeaderSize)
				binary.LittleEndian.PutUint64(b[0:8], 8)    // unpacked size
				binary.LittleEndian.PutUint64(b[8:16], 100) // packed size (too large)
				return b
			}(),
			wantErr: true,
		},
		{
			name: "Error: unpacked size not 8-byte aligned",
			packed: func() []byte {
				b := make([]byte, HeaderSize+2)
				binary.LittleEndian.PutUint64(b[0:8], 7) // not aligned
				binary.LittleEndian.PutUint64(b[8:16], 2)
				return b
			}(),
			wantErr: true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()

		_, err := Unpack(ctx, test.packed)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("[TestUnpackErrors(%s)]: got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("[TestUnpackErrors(%s)]: got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestSpecialTags(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test 0x00 tag: consecutive zero words
	t.Run("zero_words_run", func(t *testing.T) {
		// 300 zero words should use the 0x00 tag with count
		input := make([]byte, 300*8)

		packed, err := Pack(ctx, input)
		if err != nil {
			t.Fatalf("[TestSpecialTags/zero_words_run]: Pack failed: %s", err)
		}
		defer packed.Release(ctx)

		// Should be very compressed: header + a few tag+count pairs
		if packed.Len() > HeaderSize+20 {
			t.Errorf("[TestSpecialTags/zero_words_run]: packed size %d too large for %d zero bytes",
				packed.Len(), len(input))
		}

		unpacked, err := Unpack(ctx, packed.Bytes())
		if err != nil {
			t.Fatalf("[TestSpecialTags/zero_words_run]: Unpack failed: %s", err)
		}
		defer unpacked.Release(ctx)

		if !bytes.Equal(input, unpacked.Bytes()) {
			t.Error("[TestSpecialTags/zero_words_run]: roundtrip mismatch")
		}
	})

	// Test 0xFF tag: literal words run
	t.Run("literal_words_run", func(t *testing.T) {
		// All non-zero bytes, only 1 zero per word (won't trigger packing benefit)
		input := make([]byte, 32*8)
		for i := range input {
			if i%8 != 7 { // one zero per word
				input[i] = byte(i%255 + 1)
			}
		}

		packed, err := Pack(ctx, input)
		if err != nil {
			t.Fatalf("[TestSpecialTags/literal_words_run]: Pack failed: %s", err)
		}
		defer packed.Release(ctx)

		unpacked, err := Unpack(ctx, packed.Bytes())
		if err != nil {
			t.Fatalf("[TestSpecialTags/literal_words_run]: Unpack failed: %s", err)
		}
		defer unpacked.Release(ctx)

		if !bytes.Equal(input, unpacked.Bytes()) {
			t.Error("[TestSpecialTags/literal_words_run]: roundtrip mismatch")
		}
	})
}

// BenchmarkPack benchmarks the Pack function.
func BenchmarkPack(b *testing.B) {
	ctx := context.Background()

	// Simulate typical Claw message with headers
	input := make([]byte, 1024)
	for i := 0; i < len(input); i += 8 {
		input[i] = byte(i/8 + 1) // field number
		input[i+2] = 0x04        // int32 type
		input[i+3] = byte(i)     // small value
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		packed, err := Pack(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
		packed.Release(ctx)
	}
}

// BenchmarkUnpack benchmarks the Unpack function.
func BenchmarkUnpack(b *testing.B) {
	ctx := context.Background()

	// Simulate typical Claw message with headers
	input := make([]byte, 1024)
	for i := 0; i < len(input); i += 8 {
		input[i] = byte(i/8 + 1)
		input[i+2] = 0x04
		input[i+3] = byte(i)
	}

	packed, err := Pack(ctx, input)
	if err != nil {
		b.Fatal(err)
	}
	defer packed.Release(ctx)

	packedData := make([]byte, packed.Len())
	copy(packedData, packed.Bytes())

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		unpacked, err := Unpack(ctx, packedData)
		if err != nil {
			b.Fatal(err)
		}
		unpacked.Release(ctx)
	}
}

// BenchmarkRoundtrip benchmarks a full pack/unpack cycle.
func BenchmarkRoundtrip(b *testing.B) {
	ctx := context.Background()

	input := make([]byte, 1024)
	for i := 0; i < len(input); i += 8 {
		input[i] = byte(i/8 + 1)
		input[i+2] = 0x04
		input[i+3] = byte(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		packed, err := Pack(ctx, input)
		if err != nil {
			b.Fatal(err)
		}

		unpacked, err := Unpack(ctx, packed.Bytes())
		if err != nil {
			b.Fatal(err)
		}

		unpacked.Release(ctx)
		packed.Release(ctx)
	}
}
