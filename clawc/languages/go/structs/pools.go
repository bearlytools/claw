package structs

import (
	"bytes"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/values/sizes"
)

// structPool is a pool of Struct objects to reduce allocations.
var structPool = sync.NewPool[*Struct](
	context.Background(),
	"structPool",
	func() *Struct {
		return &Struct{header: NewGenericHeader()}
	},
)

var diffBuffers = newDiffSizePools()

var readers = sync.NewPool[*bytes.Reader](
	context.Background(),
	"readersPool",
	func() *bytes.Reader {
		return &bytes.Reader{}
	},
	sync.WithBuffer(100),
)

// These are all pools related to the types supported in Structs fields.
var (
	boolPool     *sync.Pool[*Bools]
	nUint8Pool   *sync.Pool[*Numbers[uint8]]
	nUint16Pool  *sync.Pool[*Numbers[uint16]]
	nUint32Pool  *sync.Pool[*Numbers[uint32]]
	nUint64Pool  *sync.Pool[*Numbers[uint64]]
	nInt8Pool    *sync.Pool[*Numbers[int8]]
	nInt16Pool   *sync.Pool[*Numbers[int16]]
	nInt32Pool   *sync.Pool[*Numbers[int32]]
	nInt64Pool   *sync.Pool[*Numbers[int64]]
	nFloat32Pool *sync.Pool[*Numbers[float32]]
	nFloat64Pool *sync.Pool[*Numbers[float64]]
	bytesPool    *sync.Pool[*Bytes]
)

// This init function initializes all the pools supporting Struct field types.
func init() {
	ctx := context.Background()

	boolPool = sync.NewPool[*Bools](
		ctx, "boolPool",
		func() *Bools { return &Bools{} },
		sync.WithBuffer(100),
	)

	nUint8Pool = sync.NewPool[*Numbers[uint8]](
		ctx, "nUint8Pool",
		func() *Numbers[uint8] { return &Numbers[uint8]{} },
		sync.WithBuffer(100),
	)

	nUint16Pool = sync.NewPool[*Numbers[uint16]](
		ctx, "nUint16Pool",
		func() *Numbers[uint16] { return &Numbers[uint16]{} },
		sync.WithBuffer(100),
	)

	nUint32Pool = sync.NewPool[*Numbers[uint32]](
		ctx, "nUint32Pool",
		func() *Numbers[uint32] { return &Numbers[uint32]{} },
		sync.WithBuffer(100),
	)

	nUint64Pool = sync.NewPool[*Numbers[uint64]](
		ctx, "nUint64Pool",
		func() *Numbers[uint64] { return &Numbers[uint64]{} },
		sync.WithBuffer(100),
	)

	nInt8Pool = sync.NewPool[*Numbers[int8]](
		ctx, "nInt8Pool",
		func() *Numbers[int8] { return &Numbers[int8]{} },
		sync.WithBuffer(100),
	)

	nInt16Pool = sync.NewPool[*Numbers[int16]](
		ctx, "nInt16Pool",
		func() *Numbers[int16] { return &Numbers[int16]{} },
		sync.WithBuffer(100),
	)

	nInt32Pool = sync.NewPool[*Numbers[int32]](
		ctx, "nInt32Pool",
		func() *Numbers[int32] { return &Numbers[int32]{} },
		sync.WithBuffer(100),
	)

	nInt64Pool = sync.NewPool[*Numbers[int64]](
		ctx, "nInt64Pool",
		func() *Numbers[int64] { return &Numbers[int64]{} },
		sync.WithBuffer(100),
	)

	nFloat32Pool = sync.NewPool[*Numbers[float32]](
		ctx, "nFloat32Pool",
		func() *Numbers[float32] { return &Numbers[float32]{} },
		sync.WithBuffer(100),
	)

	nFloat64Pool = sync.NewPool[*Numbers[float64]](
		ctx, "nFloat64Pool",
		func() *Numbers[float64] { return &Numbers[float64]{} },
		sync.WithBuffer(100),
	)

	bytesPool = sync.NewPool[*Bytes](
		ctx, "bytesPool",
		func() *Bytes { return &Bytes{} },
		sync.WithBuffer(100),
	)

}

type diffSizePools struct {
	_4B   *sync.Pool[*[]byte]
	_8B   *sync.Pool[*[]byte]
	_256B *sync.Pool[*[]byte]
	_512B *sync.Pool[*[]byte]
	_1K   *sync.Pool[*[]byte]
	_4K   *sync.Pool[*[]byte]
	_16K  *sync.Pool[*[]byte]
	_64K  *sync.Pool[*[]byte]
	_256K *sync.Pool[*[]byte]
	_1M   *sync.Pool[*[]byte]
}

func newDiffSizePools() diffSizePools {
	return diffSizePools{
		_4B: sync.NewPool(
			context.Background(),
			"diffPool4",
			func() *[]byte {
				b := make([]byte, 4)
				return &b
			},
		),
		_8B: sync.NewPool(
			context.Background(),
			"diffPool8",
			func() *[]byte {
				b := make([]byte, 8)
				return &b
			},
		),
		_256B: sync.NewPool(
			context.Background(),
			"diffPool256",
			func() *[]byte {
				b := make([]byte, 256)
				return &b
			},
		),
		_512B: sync.NewPool(
			context.Background(),
			"diffPool512",
			func() *[]byte {
				b := make([]byte, 512)
				return &b
			},
		),
		_1K: sync.NewPool(
			context.Background(),
			"diffPool1KB",
			func() *[]byte {
				b := make([]byte, 1*sizes.KiB)
				return &b
			},
		),
		_4K: sync.NewPool(
			context.Background(),
			"diffPool4K",
			func() *[]byte {
				b := make([]byte, 4*sizes.KiB)
				return &b
			},
		),
		_16K: sync.NewPool(
			context.Background(),
			"diffPool16K",
			func() *[]byte {
				b := make([]byte, 16*sizes.KiB)
				return &b
			},
		),
		_64K: sync.NewPool(
			context.Background(),
			"diffPool64K",
			func() *[]byte {
				b := make([]byte, 64*sizes.KiB)
				return &b
			},
		),
		_256K: sync.NewPool(
			context.Background(),
			"diffPool256K",
			func() *[]byte {
				b := make([]byte, 256*sizes.KiB)
				return &b
			},
		),
		_1M: sync.NewPool(
			context.Background(),
			"diffPool1M",
			func() *[]byte {
				b := make([]byte, 1*sizes.MiB)
				return &b
			},
		),
	}
}

// Get pulls a slice that will hold sizeBytes. The len/cap may be larger than requested, so
// make sure to adjust to your needs.
func (d *diffSizePools) Get(ctx context.Context, sizeBytes int) []byte {
	switch {
	case sizeBytes <= 4:
		b := d._4B.Get(ctx)
		return *b
	case sizeBytes <= 8:
		b := d._8B.Get(ctx)
		return *b
	case sizeBytes <= 256:
		b := d._256B.Get(ctx)
		return *b
	case sizeBytes <= 512:
		b := d._512B.Get(ctx)
		return *b
	case sizeBytes <= 1*sizes.KiB:
		b := d._1K.Get(ctx)
		return *b
	case sizeBytes <= 4*sizes.KiB:
		b := d._4K.Get(ctx)
		return *b
	case sizeBytes <= 16*sizes.KiB:
		b := d._16K.Get(ctx)
		return *b
	case sizeBytes <= 64*sizes.KiB:
		b := d._64K.Get(ctx)
		return *b
	case sizeBytes <= 256*sizes.KiB:
		b := d._256K.Get(ctx)
		return *b
	case sizeBytes <= 1*sizes.MiB:
		b := d._1M.Get(ctx)
		return *b
	default:
		return make([]byte, sizeBytes)
	}
}

// Put puts a []byte into a pool for reuse.
func (d *diffSizePools) Put(ctx context.Context, b []byte) {
	switch {
	case cap(b) < 8:
		d._4B.Put(ctx, &b)
	// I want to get rid of everything > 16 and < 256.
	case cap(b) == 8 || cap(b) <= 16:
		d._8B.Put(ctx, &b)
	case cap(b) < 256:
		// Do not pool slices between 16 and 256 bytes to reduce fragmentation.
	case cap(b) < 512:
		d._256B.Put(ctx, &b)
	case cap(b) < 1*sizes.KiB:
		d._512B.Put(ctx, &b)
	case cap(b) < 4*sizes.KiB:
		d._1K.Put(ctx, &b)
	case cap(b) < 16*sizes.KiB:
		d._4K.Put(ctx, &b)
	case cap(b) < 64*sizes.KiB:
		d._16K.Put(ctx, &b)
	case cap(b) < 256*sizes.KiB:
		d._64K.Put(ctx, &b)
	case cap(b) < 1*sizes.MiB:
		d._256K.Put(ctx, &b)
	case cap(b) == 1*sizes.MiB:
		d._1M.Put(ctx, &b)
	default:
		// Larger than 1MiB - don't pool, let GC handle it
	}
}
