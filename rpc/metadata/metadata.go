// Package metadata provides types for handling RPC request and response metadata.
// Metadata allows passing additional information alongside RPC calls, similar to
// HTTP headers.
package metadata

import (
	"strings"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// MD is a mapping from metadata keys to values. Keys are case-insensitive.
type MD map[string][]byte

// New creates metadata from key-value pairs.
// Pairs must be provided as (key, value, key, value, ...).
// Values are converted to []byte.
func New(kv ...string) MD {
	if len(kv)%2 != 0 {
		panic("metadata: New requires even number of arguments")
	}
	md := make(MD, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key := strings.ToLower(kv[i])
		md[key] = []byte(kv[i+1])
	}
	return md
}

// Pairs creates metadata from key-value byte pairs.
// Pairs must be provided as (key, value, key, value, ...).
func Pairs(kv ...any) MD {
	if len(kv)%2 != 0 {
		panic("metadata: Pairs requires even number of arguments")
	}
	md := make(MD, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			panic("metadata: Pairs key must be string")
		}
		key = strings.ToLower(key)
		switch v := kv[i+1].(type) {
		case string:
			md[key] = []byte(v)
		case []byte:
			md[key] = v
		default:
			panic("metadata: Pairs value must be string or []byte")
		}
	}
	return md
}

// Get retrieves a metadata value by key. Keys are case-insensitive.
// Returns nil if the key does not exist.
func (md MD) Get(key string) []byte {
	return md[strings.ToLower(key)]
}

// GetString retrieves a metadata value as a string. Keys are case-insensitive.
// Returns empty string if the key does not exist.
func (md MD) GetString(key string) string {
	if v := md[strings.ToLower(key)]; v != nil {
		return string(v)
	}
	return ""
}

// Set sets a metadata key to a value. Keys are case-insensitive.
func (md MD) Set(key string, value []byte) {
	md[strings.ToLower(key)] = value
}

// SetString sets a metadata key to a string value. Keys are case-insensitive.
func (md MD) SetString(key, value string) {
	md[strings.ToLower(key)] = []byte(value)
}

// Delete removes a metadata key. Keys are case-insensitive.
func (md MD) Delete(key string) {
	delete(md, strings.ToLower(key))
}

// Clone returns a copy of the metadata.
func (md MD) Clone() MD {
	if md == nil {
		return nil
	}
	clone := make(MD, len(md))
	for k, v := range md {
		vCopy := make([]byte, len(v))
		copy(vCopy, v)
		clone[k] = vCopy
	}
	return clone
}

// Len returns the number of metadata entries.
func (md MD) Len() int {
	return len(md)
}

// FromMsgs converts a slice of internal msgs.Metadata to MD.
func FromMsgs(ctx context.Context, mds []msgs.Metadata) MD {
	if len(mds) == 0 {
		return nil
	}
	md := make(MD, len(mds))
	for _, m := range mds {
		key := strings.ToLower(m.Key())
		md[key] = m.Value()
	}
	return md
}

// ToMsgs converts MD to a slice of internal msgs.Metadata.
func (md MD) ToMsgs(ctx context.Context) []msgs.Metadata {
	if len(md) == 0 {
		return nil
	}
	mds := make([]msgs.Metadata, 0, len(md))
	for k, v := range md {
		m := msgs.NewMetadata(ctx)
		m.SetKey(k)
		m.SetValue(v)
		mds = append(mds, m)
	}
	return mds
}

// mdKey is the context key for metadata.
type mdKey struct{}

// NewContext creates a new context with the provided metadata attached.
func NewContext(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, mdKey{}, md)
}

// FromContext retrieves metadata from a context.
// Returns nil, false if no metadata is attached.
func FromContext(ctx context.Context) (MD, bool) {
	md, ok := ctx.Value(mdKey{}).(MD)
	return md, ok
}

// AppendToContext appends key-value pairs to metadata in the context.
// If no metadata exists, creates new metadata.
// Pairs must be provided as (key, value, key, value, ...).
func AppendToContext(ctx context.Context, kv ...string) context.Context {
	md, ok := FromContext(ctx)
	if !ok {
		md = New(kv...)
	} else {
		// Clone to avoid modifying the original.
		md = md.Clone()
		for i := 0; i < len(kv); i += 2 {
			md.SetString(kv[i], kv[i+1])
		}
	}
	return NewContext(ctx, md)
}
