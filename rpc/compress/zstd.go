package compress

import (
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/klauspost/compress/zstd"
)

// Zstd implements Compressor using the Zstandard compression algorithm.
// Zstd provides excellent compression ratios with good speed.
type Zstd struct {
	// Level is the compression level. Use zstd.SpeedFastest, zstd.SpeedDefault,
	// zstd.SpeedBetterCompression, or zstd.SpeedBestCompression.
	// If 0, defaults to zstd.SpeedDefault.
	Level zstd.EncoderLevel
}

// Type returns the compression type for the wire protocol.
func (z *Zstd) Type() msgs.Compression {
	return msgs.CmpZstd
}

// Compress compresses data using Zstandard.
func (z *Zstd) Compress(data []byte) ([]byte, error) {
	level := z.Level
	if level == 0 {
		level = zstd.SpeedDefault
	}
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
	if err != nil {
		return nil, err
	}
	defer enc.Close()
	return enc.EncodeAll(data, nil), nil
}

// Decompress decompresses Zstandard data.
func (z *Zstd) Decompress(data []byte) ([]byte, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	return dec.DecodeAll(data, nil)
}
