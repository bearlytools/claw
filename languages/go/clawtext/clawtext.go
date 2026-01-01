package clawtext

import (
	"bytes"
	"io"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/values/sizes"
)

// Ingester is an interface for ingesting Claw tokens into a struct.
type Ingester interface {
	Ingest(context.Context, clawiter.Walker, ...clawiter.IngestOption) error
}

var marshalPool = &marshallerPool{
	pool: sync.NewPool[*bytes.Buffer](
		context.Background(),
		"clawtext.marshallerPool",
		func() *bytes.Buffer {
			b := &bytes.Buffer{}
			b.Grow(256)
			return b
		},
	),
}

// Buffer is a bytes.Buffer with a Release method to return it to the pool.
type Buffer struct {
	*bytes.Buffer
}

// Release returns the Buffer to the pool. Only use this once you are done with it.
func (b Buffer) Release(ctx context.Context) {
	marshalPool.put(ctx, b.Buffer)
}

type marshallerPool struct {
	pool *sync.Pool[*bytes.Buffer]
}

func (m *marshallerPool) get(ctx context.Context) *bytes.Buffer {
	return m.pool.Get(ctx)
}

func (m *marshallerPool) put(ctx context.Context, b *bytes.Buffer) {
	if b.Cap() > 10*sizes.MiB {
		return
	}
	b.Reset()
	m.pool.Put(ctx, b)
}

// Marshal marshals the Walkable to clawtext format.
func Marshal(ctx context.Context, v Walkable, options ...MarshalOption) (Buffer, error) {
	buf := marshalPool.get(ctx)
	if err := MarshalWriter(ctx, v, buf, options...); err != nil {
		return Buffer{}, err
	}
	return Buffer{buf}, nil
}

// MarshalWriter marshals the Walkable to clawtext, writing to the provided io.Writer.
func MarshalWriter(ctx context.Context, v Walkable, w io.Writer, options ...MarshalOption) error {
	opts := marshalOptions{}
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return err
		}
	}
	return writeClawtext(ctx, w, v, opts)
}

// unmarshalOptions provides options for reading clawtext into Claw structs.
type unmarshalOptions struct {
	IgnoreUnknownFields bool
}

// UnmarshalOption provides options for unmarshaling clawtext to Claw.
type UnmarshalOption func(unmarshalOptions) (unmarshalOptions, error)

// WithIgnoreUnknownFields configures whether unknown fields should be ignored.
func WithIgnoreUnknownFields(ignore bool) UnmarshalOption {
	return func(u unmarshalOptions) (unmarshalOptions, error) {
		u.IgnoreUnknownFields = ignore
		return u, nil
	}
}

// Unmarshal parses clawtext data and populates the Ingester.
func Unmarshal(ctx context.Context, data []byte, v Ingester, options ...UnmarshalOption) error {
	return UnmarshalReader(ctx, bytes.NewReader(data), v, options...)
}

// UnmarshalReader parses clawtext from a reader and populates the Ingester.
func UnmarshalReader(ctx context.Context, r io.Reader, v Ingester, options ...UnmarshalOption) error {
	opts := unmarshalOptions{}
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return err
		}
	}

	// Read all input into a string for halfpike
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return err
	}

	tokens := textToTokens(buf.String())
	walker := clawiter.Walker(func(yield clawiter.YieldToken) {
		tokens(yield)
	})

	var ingestOpts []clawiter.IngestOption
	if opts.IgnoreUnknownFields {
		ingestOpts = append(ingestOpts, clawiter.WithIgnoreUnknownFields(true))
	}
	return v.Ingest(ctx, walker, ingestOpts...)
}
