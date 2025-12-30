package compress

import (
	"bytes"
	"testing"

	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/kylelemons/godebug/pretty"
)

func TestCompressors(t *testing.T) {
	tests := []struct {
		name string
		alg  msgs.Compression
		data []byte
	}{
		{"Success: gzip small data", msgs.CmpGzip, []byte("hello world")},
		{"Success: gzip large data", msgs.CmpGzip, bytes.Repeat([]byte("hello world "), 1000)},
		{"Success: snappy small data", msgs.CmpSnappy, []byte("hello world")},
		{"Success: snappy large data", msgs.CmpSnappy, bytes.Repeat([]byte("hello world "), 1000)},
		{"Success: zstd small data", msgs.CmpZstd, []byte("hello world")},
		{"Success: zstd large data", msgs.CmpZstd, bytes.Repeat([]byte("hello world "), 1000)},
		{"Success: none passthrough", msgs.CmpNone, []byte("hello world")},
	}

	for _, test := range tests {
		compressed, err := Compress(test.alg, test.data)
		switch {
		case err != nil:
			t.Errorf("TestCompressors(%s): Compress got err == %s, want err == nil", test.name, err)
			continue
		}

		decompressed, err := Decompress(test.alg, compressed)
		switch {
		case err != nil:
			t.Errorf("TestCompressors(%s): Decompress got err == %s, want err == nil", test.name, err)
			continue
		}

		if diff := pretty.Compare(test.data, decompressed); diff != "" {
			t.Errorf("TestCompressors(%s): roundtrip mismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestCompressEmptyData(t *testing.T) {
	tests := []struct {
		name string
		alg  msgs.Compression
	}{
		{"Success: gzip empty", msgs.CmpGzip},
		{"Success: snappy empty", msgs.CmpSnappy},
		{"Success: zstd empty", msgs.CmpZstd},
		{"Success: none empty", msgs.CmpNone},
	}

	for _, test := range tests {
		compressed, err := Compress(test.alg, nil)
		switch {
		case err != nil:
			t.Errorf("TestCompressEmptyData(%s): Compress got err == %s, want err == nil", test.name, err)
			continue
		}

		decompressed, err := Decompress(test.alg, compressed)
		switch {
		case err != nil:
			t.Errorf("TestCompressEmptyData(%s): Decompress got err == %s, want err == nil", test.name, err)
			continue
		}

		if len(decompressed) != 0 {
			t.Errorf("TestCompressEmptyData(%s): got len %d, want 0", test.name, len(decompressed))
		}
	}
}

func TestCompressActuallyCompresses(t *testing.T) {
	// Test that compression actually reduces size for compressible data.
	data := bytes.Repeat([]byte("hello world "), 1000) // 12000 bytes of repetitive data

	tests := []struct {
		name string
		alg  msgs.Compression
	}{
		{"Success: gzip compresses", msgs.CmpGzip},
		{"Success: snappy compresses", msgs.CmpSnappy},
		{"Success: zstd compresses", msgs.CmpZstd},
	}

	for _, test := range tests {
		compressed, err := Compress(test.alg, data)
		switch {
		case err != nil:
			t.Errorf("TestCompressActuallyCompresses(%s): got err == %s, want err == nil", test.name, err)
			continue
		}

		if len(compressed) >= len(data) {
			t.Errorf("TestCompressActuallyCompresses(%s): compressed size %d >= original size %d", test.name, len(compressed), len(data))
		}
	}
}

func TestCustomCompressor(t *testing.T) {
	// Test that custom compressors can be registered and used.
	custom := &testCompressor{}
	Register(custom)

	data := []byte("test data")
	compressed, err := Compress(msgs.Compression(100), data)
	switch {
	case err != nil:
		t.Errorf("TestCustomCompressor: Compress got err == %s, want err == nil", err)
		return
	}

	decompressed, err := Decompress(msgs.Compression(100), compressed)
	switch {
	case err != nil:
		t.Errorf("TestCustomCompressor: Decompress got err == %s, want err == nil", err)
		return
	}

	if diff := pretty.Compare(data, decompressed); diff != "" {
		t.Errorf("TestCustomCompressor: roundtrip mismatch (-want +got):\n%s", diff)
	}
}

func TestUnregisteredCompressor(t *testing.T) {
	// Test that unregistered compressor returns error.
	_, err := Compress(msgs.Compression(200), []byte("data"))
	if err == nil {
		t.Errorf("TestUnregisteredCompressor: Compress got err == nil, want err != nil")
	}

	_, err = Decompress(msgs.Compression(200), []byte("data"))
	if err == nil {
		t.Errorf("TestUnregisteredCompressor: Decompress got err == nil, want err != nil")
	}
}

func TestGetCompressor(t *testing.T) {
	tests := []struct {
		name    string
		alg     msgs.Compression
		wantNil bool
	}{
		{"Success: get gzip", msgs.CmpGzip, false},
		{"Success: get snappy", msgs.CmpSnappy, false},
		{"Success: get zstd", msgs.CmpZstd, false},
		{"Success: get none returns nil", msgs.CmpNone, true},
		{"Success: get unregistered returns nil", msgs.Compression(250), true},
	}

	for _, test := range tests {
		c := Get(test.alg)
		switch {
		case test.wantNil && c != nil:
			t.Errorf("TestGetCompressor(%s): got compressor, want nil", test.name)
		case !test.wantNil && c == nil:
			t.Errorf("TestGetCompressor(%s): got nil, want compressor", test.name)
		}
	}
}

// testCompressor is a simple compressor for testing custom registration.
type testCompressor struct{}

func (t *testCompressor) Type() msgs.Compression { return msgs.Compression(100) }

func (t *testCompressor) Compress(data []byte) ([]byte, error) {
	// Simple "compression": just reverse the bytes
	result := make([]byte, len(data))
	for i, b := range data {
		result[len(data)-1-i] = b
	}
	return result, nil
}

func (t *testCompressor) Decompress(data []byte) ([]byte, error) {
	// "Decompress": reverse back
	result := make([]byte, len(data))
	for i, b := range data {
		result[len(data)-1-i] = b
	}
	return result, nil
}
