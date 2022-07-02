package structs

import (
	"bytes"
	"sync"

	autopool "github.com/johnsiilver/golib/development/autopool/blend"
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

var readers = sync.Pool{
	New: func() any {
		return &bytes.Reader{}
	},
}

var (
	pool         = autopool.New()
	boolPool     int
	nUint8Pool   int
	nUint16Pool  int
	nUint32Pool  int
	nUint64Pool  int
	nInt8Pool    int
	nInt16Pool   int
	nInt32Pool   int
	nInt64Pool   int
	nFloat32Pool int
	nFloat64Pool int
	bytesPool    int
)

func init() {
	boolPool = pool.Add(
		func() any {
			return &Bool{}
		},
	)

	nUint8Pool = pool.Add(
		func() any {
			return &Number[uint8]{}
		},
	)

	nUint16Pool = pool.Add(
		func() any {
			return &Number[uint16]{}
		},
	)

	nUint32Pool = pool.Add(
		func() any {
			return &Number[uint32]{}
		},
	)

	nUint64Pool = pool.Add(
		func() any {
			return &Number[uint64]{}
		},
	)

	nInt8Pool = pool.Add(
		func() any {
			return &Number[int8]{}
		},
	)

	nInt16Pool = pool.Add(
		func() any {
			return &Number[int16]{}
		},
	)

	nInt32Pool = pool.Add(
		func() any {
			return &Number[int32]{}
		},
	)

	nInt64Pool = pool.Add(
		func() any {
			return &Number[int64]{}
		},
	)

	nFloat32Pool = pool.Add(
		func() any {
			return &Number[float32]{}
		},
	)

	nFloat64Pool = pool.Add(
		func() any {
			return &Number[float64]{}
		},
	)

	bytesPool = pool.Add(
		func() any {
			return &Bytes{}
		},
	)
}

/*
// pool holds a few sync.Pool(s) that we can use for buffer reuse.
var pool = &pools{
	_32: &sync.Pool{
		New: func() any {
			return make([]byte, 4)
		},
	},
	_64: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 8)
		},
	},
	_128: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 16)
		},
	},
	buff: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 64)
		},
	},
}

type pools struct {
	_32  *sync.Pool
	_64  *sync.Pool
	_128 *sync.Pool
	buff *sync.Pool
}

func (p *pools) get32() []byte {
	return p._32.Get().([]byte)
}

func (p *pools) get64() []byte {
	return p._64.Get().([]byte)
}

func (p *pools) get128() []byte {
	return p._128.Get().([]byte)
}

func (p *pools) getBuff() []byte {
	return p.buff.Get().([]byte)
}

func (p *pools) put(b []byte) {
	switch len(b) {
	case 32:
		p._32.Put(b)
	case 64:
		p._64.Put(b)
	case 128:
		p._128.Put(b)
	default:
		p.buff.Put(b)
	}
}
*/
