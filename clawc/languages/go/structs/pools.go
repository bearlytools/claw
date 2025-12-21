package structs

import (
	"bytes"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
)

var ()

type BytesPool interface {
	Get() *[]byte
	Put(*[]byte)
}

type StructsPool interface {
	Get() *Struct
	Put(*Struct)
}

var readers = sync.NewPool[*bytes.Reader](
	context.Background(),
	"readersPool",
	func() *bytes.Reader {
		return &bytes.Reader{}
	},
	sync.WithBuffer(100),
)

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

