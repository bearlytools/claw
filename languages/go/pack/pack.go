// Package pack implements Cap'n Proto-style packing for Claw messages.
//
// Packing compresses messages by eliminating zero bytes. Each 8-byte word
// is reduced to a tag byte followed by only the non-zero bytes from that word.
// The tag byte's bits correspond to the bytes of the word (LSB = first byte).
// A set bit means that byte is non-zero and present; a clear bit means zero.
//
// Special cases:
//   - Tag 0x00: Followed by a count byte indicating additional all-zero words (0-255)
//   - Tag 0xFF: Followed by 8 literal bytes, then a count of additional literal words
//
// Wire format:
//
//	+------------------+------------------+------------------+
//	| Unpacked Size    | Packed Size      | Packed Data      |
//	| (8 bytes LE)     | (8 bytes LE)     | (variable)       |
//	+------------------+------------------+------------------+
//
// This implementation is zero-allocation in the hot path. Buffers are pooled
// and reused. Call [Buffer.Release] when done with a buffer.
package pack

import (
	"encoding/binary"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/languages/go/errors"
)

const (
	// HeaderSize is the size of the pack header (unpacked size + packed size).
	HeaderSize = 16
)

// Buffer wraps a byte slice with a Release method to return it to the pool.
type Buffer struct {
	data []byte
}

// Bytes returns the underlying byte slice.
func (b *Buffer) Bytes() []byte {
	return b.data
}

// Len returns the length of the buffer.
func (b *Buffer) Len() int {
	return len(b.data)
}

// Release returns the buffer to the pool for reuse.
// The buffer must not be used after calling Release.
func (b *Buffer) Release(ctx context.Context) {
	if b == nil {
		return
	}
	bufferPool.Put(ctx, b)
}

// Reset implements the Resetter interface for sync.Pool.
func (b *Buffer) Reset() {
	b.data = b.data[:0]
}

// bufferPool provides pooled buffers to reduce allocations.
var bufferPool = sync.NewPool[*Buffer](
	context.Background(),
	"pack.bufferPool",
	func() *Buffer {
		return &Buffer{
			data: make([]byte, 0, 4096),
		}
	},
)

// maxPackedSize returns the maximum possible size for packed output (excluding header).
func maxPackedSize(unpackedLen int) int {
	if unpackedLen == 0 {
		return 0
	}
	words := unpackedLen / 8
	// Each word: 1 tag byte + 0-8 data bytes
	// For 0xFF tag: +1 count byte per run (amortized over up to 256 words)
	// Safe upper bound: original + 1 per word + 1 per 256 words
	return unpackedLen + words + (words+255)/256
}

// Pack compresses src into a pooled buffer using Cap'n Proto packing.
// src must be 8-byte aligned (len(src) % 8 == 0).
// Returns a Buffer that must be released via [Buffer.Release] when done.
//
// This function performs zero heap allocations in the steady state.
func Pack(ctx context.Context, src []byte) (*Buffer, error) {
	srcLen := len(src)
	if srcLen%8 != 0 {
		return nil, errors.New("pack: input size must be divisible by 8")
	}
	if srcLen == 0 {
		return nil, nil
	}

	// Get a buffer from the pool
	buf := bufferPool.Get(ctx)

	// Ensure capacity for header + max packed size
	needed := HeaderSize + maxPackedSize(srcLen)
	if cap(buf.data) < needed {
		buf.data = make([]byte, needed)
	} else {
		buf.data = buf.data[:needed]
	}

	// Reserve space for header, pack into the rest
	packedLen, err := packInto(buf.data[HeaderSize:], src)
	if err != nil {
		buf.Release(ctx)
		return nil, err
	}

	// Write header
	binary.LittleEndian.PutUint64(buf.data[0:8], uint64(srcLen))
	binary.LittleEndian.PutUint64(buf.data[8:16], uint64(packedLen))

	// Trim to actual size
	buf.data = buf.data[:HeaderSize+packedLen]

	return buf, nil
}

// packInto packs src into dst, returning the number of bytes written.
// dst must be at least maxPackedSize(len(src)) bytes.
func packInto(dst, src []byte) (int, error) {
	w := 0 // write position in dst
	r := 0 // read position in src
	srcLen := len(src)

	for r < srcLen {
		word := binary.LittleEndian.Uint64(src[r:])

		// Compute tag: bit i is set if byte i is non-zero
		tag := computeTag(word)
		dst[w] = tag
		w++

		// Write non-zero bytes
		w += packWord(dst[w:], word, tag)
		r += 8

		switch tag {
		case 0x00:
			// Count additional consecutive zero words (up to 255)
			count := byte(0)
			for r < srcLen && count < 255 {
				if binary.LittleEndian.Uint64(src[r:]) != 0 {
					break
				}
				count++
				r += 8
			}
			dst[w] = count
			w++

		case 0xFF:
			// Count additional literal words that don't benefit from packing
			countPos := w
			w++ // reserve space for count byte
			count := byte(0)

			for r < srcLen && count < 255 {
				word := binary.LittleEndian.Uint64(src[r:])
				// If 2+ zeros, packing saves space; end literal run
				if countZerosInWord(word) >= 2 {
					break
				}
				binary.LittleEndian.PutUint64(dst[w:], word)
				w += 8
				r += 8
				count++
			}
			dst[countPos] = count
		}
	}

	return w, nil
}

// Unpack decompresses packed data into a pooled buffer.
// Returns a Buffer that must be released via [Buffer.Release] when done.
//
// This function performs zero heap allocations in the steady state.
func Unpack(ctx context.Context, packed []byte) (*Buffer, error) {
	if len(packed) < HeaderSize {
		return nil, errors.New("pack: data too short for header")
	}

	unpackedSize := int(binary.LittleEndian.Uint64(packed[0:8]))
	packedSize := int(binary.LittleEndian.Uint64(packed[8:16]))

	if len(packed) < HeaderSize+packedSize {
		return nil, errors.New("pack: data shorter than declared packed size")
	}

	if unpackedSize%8 != 0 {
		return nil, errors.New("pack: invalid unpacked size (not 8-byte aligned)")
	}

	// Get a buffer from the pool
	buf := bufferPool.Get(ctx)

	// Ensure capacity
	if cap(buf.data) < unpackedSize {
		buf.data = make([]byte, unpackedSize)
	} else {
		buf.data = buf.data[:unpackedSize]
	}

	// Unpack
	n, err := unpackInto(buf.data, packed[HeaderSize:HeaderSize+packedSize])
	if err != nil {
		buf.Release(ctx)
		return nil, err
	}

	if n != unpackedSize {
		buf.Release(ctx)
		return nil, errors.New("pack: unpacked size mismatch")
	}

	return buf, nil
}

// unpackInto unpacks src into dst, returning the number of bytes written.
func unpackInto(dst, src []byte) (int, error) {
	w := 0 // write position in dst
	r := 0 // read position in src
	srcLen := len(src)
	dstLen := len(dst)

	for r < srcLen {
		if w+8 > dstLen {
			return 0, errors.New("pack: output buffer too small")
		}

		tag := src[r]
		r++

		// Unpack word according to tag bits
		bytesRead := unpackWord(dst[w:w+8], src[r:], tag)
		r += bytesRead
		w += 8

		switch tag {
		case 0x00:
			if r >= srcLen {
				return 0, errors.New("pack: unexpected end of packed data")
			}
			count := int(src[r])
			r++

			// Write count zero words
			zeroBytes := count * 8
			if w+zeroBytes > dstLen {
				return 0, errors.New("pack: output buffer too small")
			}
			for i := 0; i < zeroBytes; i++ {
				dst[w+i] = 0
			}
			w += zeroBytes

		case 0xFF:
			if r >= srcLen {
				return 0, errors.New("pack: unexpected end of packed data")
			}
			count := int(src[r])
			r++

			// Copy count literal words
			literalBytes := count * 8
			if r+literalBytes > srcLen {
				return 0, errors.New("pack: unexpected end of packed data")
			}
			if w+literalBytes > dstLen {
				return 0, errors.New("pack: output buffer too small")
			}
			copy(dst[w:], src[r:r+literalBytes])
			r += literalBytes
			w += literalBytes
		}
	}

	return w, nil
}

// computeTag returns a tag byte where bit i is set if byte i of word is non-zero.
func computeTag(word uint64) byte {
	var tag byte
	for i := 0; i < 8; i++ {
		if (word>>(i*8))&0xFF != 0 {
			tag |= 1 << i
		}
	}
	return tag
}

// packWord writes the non-zero bytes of word to dst according to tag.
// Returns the number of bytes written.
func packWord(dst []byte, word uint64, tag byte) int {
	n := 0
	for i := 0; i < 8; i++ {
		if tag&(1<<i) != 0 {
			dst[n] = byte(word >> (i * 8))
			n++
		}
	}
	return n
}

// unpackWord reconstructs an 8-byte word from packed bytes according to tag.
// dst must be exactly 8 bytes. src contains the packed non-zero bytes.
// Returns the number of bytes read from src.
func unpackWord(dst, src []byte, tag byte) int {
	srcIdx := 0
	for i := 0; i < 8; i++ {
		if tag&(1<<i) != 0 {
			dst[i] = src[srcIdx]
			srcIdx++
		} else {
			dst[i] = 0
		}
	}
	return srcIdx
}

// countZerosInWord counts zero bytes in a 64-bit word.
func countZerosInWord(word uint64) int {
	count := 0
	for i := 0; i < 8; i++ {
		if (word>>(i*8))&0xFF == 0 {
			count++
		}
	}
	return count
}

// UnpackedSize returns the unpacked size from the header of packed data.
// Returns 0 if the data is too short.
func UnpackedSize(packed []byte) int {
	if len(packed) < HeaderSize {
		return 0
	}
	return int(binary.LittleEndian.Uint64(packed[0:8]))
}

// PackedSize returns the packed data size from the header (excluding header).
// Returns 0 if the data is too short.
func PackedSize(packed []byte) int {
	if len(packed) < HeaderSize {
		return 0
	}
	return int(binary.LittleEndian.Uint64(packed[8:16]))
}

// CompressionRatio returns the compression ratio for the given packed data.
// Returns 0 if unpacked size is 0.
func CompressionRatio(packed []byte) float64 {
	unpacked := UnpackedSize(packed)
	if unpacked == 0 {
		return 0
	}
	return float64(len(packed)) / float64(unpacked)
}
