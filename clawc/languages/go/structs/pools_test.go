package structs

import (
	"testing"
)

func TestDiffSizePoolsGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requestSize int
		wantMinCap  int
	}{
		{
			name:        "Success: request 100 bytes gets 256B buffer",
			requestSize: 100,
			wantMinCap:  256,
		},
		{
			name:        "Success: request 256 bytes gets 256B buffer",
			requestSize: 256,
			wantMinCap:  256,
		},
		{
			name:        "Success: request 257 bytes gets 512B buffer",
			requestSize: 257,
			wantMinCap:  512,
		},
		{
			name:        "Success: request 512 bytes gets 512B buffer",
			requestSize: 512,
			wantMinCap:  512,
		},
		{
			name:        "Success: request 513 bytes gets 1K buffer",
			requestSize: 513,
			wantMinCap:  1024,
		},
		{
			name:        "Success: request 1K bytes gets 1K buffer",
			requestSize: 1024,
			wantMinCap:  1024,
		},
		{
			name:        "Success: request 2K bytes gets 4K buffer",
			requestSize: 2048,
			wantMinCap:  4096,
		},
		{
			name:        "Success: request 4K bytes gets 4K buffer",
			requestSize: 4096,
			wantMinCap:  4096,
		},
		{
			name:        "Success: request 5K bytes gets 16K buffer",
			requestSize: 5 * 1024,
			wantMinCap:  16 * 1024,
		},
		{
			name:        "Success: request 16K bytes gets 16K buffer",
			requestSize: 16 * 1024,
			wantMinCap:  16 * 1024,
		},
		{
			name:        "Success: request 20K bytes gets 64K buffer",
			requestSize: 20 * 1024,
			wantMinCap:  64 * 1024,
		},
		{
			name:        "Success: request 64K bytes gets 64K buffer",
			requestSize: 64 * 1024,
			wantMinCap:  64 * 1024,
		},
		{
			name:        "Success: request 100K bytes gets 256K buffer",
			requestSize: 100 * 1024,
			wantMinCap:  256 * 1024,
		},
		{
			name:        "Success: request 256K bytes gets 256K buffer",
			requestSize: 256 * 1024,
			wantMinCap:  256 * 1024,
		},
		{
			name:        "Success: request 500K bytes gets 1M buffer",
			requestSize: 500 * 1024,
			wantMinCap:  1024 * 1024,
		},
		{
			name:        "Success: request 1M bytes gets 1M buffer",
			requestSize: 1024 * 1024,
			wantMinCap:  1024 * 1024,
		},
		{
			name:        "Success: request 2M bytes gets exact size (not pooled)",
			requestSize: 2 * 1024 * 1024,
			wantMinCap:  2 * 1024 * 1024,
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		got := diffBuffers.Get(ctx, test.requestSize)
		if cap(got) < test.wantMinCap {
			t.Errorf("[TestDiffSizePoolsGet] %s: cap(got) = %d, want >= %d", test.name, cap(got), test.wantMinCap)
		}
	}
}

func TestDiffSizePoolsPut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bufSize  int
		putTwice bool
	}{
		{
			name:     "Success: put 256B buffer",
			bufSize:  256,
			putTwice: true,
		},
		{
			name:     "Success: put 512B buffer",
			bufSize:  512,
			putTwice: true,
		},
		{
			name:     "Success: put 1K buffer",
			bufSize:  1024,
			putTwice: true,
		},
		{
			name:     "Success: put 4K buffer",
			bufSize:  4 * 1024,
			putTwice: true,
		},
		{
			name:     "Success: put 16K buffer",
			bufSize:  16 * 1024,
			putTwice: true,
		},
		{
			name:     "Success: put 64K buffer",
			bufSize:  64 * 1024,
			putTwice: true,
		},
		{
			name:     "Success: put 256K buffer",
			bufSize:  256 * 1024,
			putTwice: true,
		},
		{
			name:     "Success: put 1M buffer",
			bufSize:  1024 * 1024,
			putTwice: true,
		},
		{
			name:     "Success: put 2M buffer (not pooled, no panic)",
			bufSize:  2 * 1024 * 1024,
			putTwice: false,
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		buf := make([]byte, test.bufSize)

		// Put should not panic
		diffBuffers.Put(ctx, buf)

		if test.putTwice {
			// Get and put again to verify reuse works
			got := diffBuffers.Get(ctx, test.bufSize)
			if cap(got) < test.bufSize {
				t.Errorf("[TestDiffSizePoolsPut] %s: after put/get, cap(got) = %d, want >= %d", test.name, cap(got), test.bufSize)
			}
			diffBuffers.Put(ctx, got)
		}
	}
}

func TestDiffSizePoolsBoundaries(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test that boundary values go to the correct pool
	boundaries := []struct {
		size       int
		expectCap  int
		desc       string
	}{
		{256, 256, "exactly 256"},
		{257, 512, "just over 256"},
		{512, 512, "exactly 512"},
		{513, 1024, "just over 512"},
		{1024, 1024, "exactly 1K"},
		{1025, 4096, "just over 1K"},
	}

	for _, b := range boundaries {
		got := diffBuffers.Get(ctx, b.size)
		if cap(got) < b.expectCap {
			t.Errorf("[TestDiffSizePoolsBoundaries] %s: cap = %d, want >= %d", b.desc, cap(got), b.expectCap)
		}
		diffBuffers.Put(ctx, got)
	}
}
