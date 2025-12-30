package compress

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Gzip implements Compressor using the gzip compression algorithm.
type Gzip struct {
	// Level is the compression level. Use gzip.DefaultCompression (0),
	// gzip.NoCompression, gzip.BestSpeed, or gzip.BestCompression.
	// If 0, defaults to gzip.DefaultCompression.
	Level int
}

// Type returns the compression type for the wire protocol.
func (g *Gzip) Type() msgs.Compression {
	return msgs.CmpGzip
}

// Compress compresses data using gzip.
func (g *Gzip) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	level := g.Level
	if level == 0 {
		level = gzip.DefaultCompression
	}
	w, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress decompresses gzip data.
func (g *Gzip) Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
