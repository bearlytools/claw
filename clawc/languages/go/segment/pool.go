package segment

import (
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/concurrency/sync"

	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// DefaultPool is the default struct pool for segment.Struct instances.
var DefaultPool *sync.Pool[*Struct]

func init() {
	DefaultPool = sync.NewPool[*Struct](
		context.Background(),
		"segment.Struct",
		func() *Struct {
			return &Struct{
				seg:        NewSegment(256),
				fieldIndex: make([]fieldEntry, 32),
				dirtyLists: make([]dirtyList, 0, 4),
			}
		},
	)
}

// NewPooled gets a struct from the pool and initializes it with the mapping.
func NewPooled(ctx context.Context, m *mapping.Map) *Struct {
	s := DefaultPool.Get(ctx)
	s.Init(m)
	return s
}

// Release returns a struct to the pool.
// The pool automatically calls s.Reset() via the Resetter interface.
func Release(ctx context.Context, s *Struct) {
	DefaultPool.Put(ctx, s)
}
