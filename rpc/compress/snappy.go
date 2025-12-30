package compress

import (
	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/golang/snappy"
)

// Snappy implements Compressor using the Snappy compression algorithm.
// Snappy is optimized for speed rather than compression ratio.
type Snappy struct{}

// Type returns the compression type for the wire protocol.
func (s *Snappy) Type() msgs.Compression {
	return msgs.CmpSnappy
}

// Compress compresses data using Snappy.
func (s *Snappy) Compress(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

// Decompress decompresses Snappy data.
func (s *Snappy) Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}
