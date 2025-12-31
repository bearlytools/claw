// Package clawjson provides functionality to marshal Claw structures to JSON.
package clawjson

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"iter"
	"math"
	"strconv"
	"unicode/utf8"
	"unsafe"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	jsonv2 "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/values/sizes"
)

// Pre-allocated byte slices for common JSON tokens to avoid allocations.
var (
	jsonOpenBrace    = []byte("{")
	jsonCloseBrace   = []byte("}")
	jsonOpenBracket  = []byte("[")
	jsonCloseBracket = []byte("]")
	jsonComma        = []byte(",")
	jsonColon        = []byte(":")
	jsonNull         = []byte("null")
	jsonTrue         = []byte("true")
	jsonFalse        = []byte("false")
	jsonQuote        = []byte(`"`)
	jsonEmptyArray   = []byte("[]")
)

// Walkable is an interface for types that can walk over Claw tokens.
type Walkable interface {
	Walk(ctx context.Context, yield clawiter.YieldToken, opts ...clawiter.WalkOption)
}

// marshalOptions provides options for writing Claw output to JSON.
type marshalOptions struct {
	UseEnumNumbers bool
}

// MarshalOption provides options for marshaling Claw to JSON.
type MarshalOption func(marshalOptions) (marshalOptions, error)

// WithUseEnumNumbers configures whether enum values are emitted as numbers or strings.
func WithUseEnumNumbers(use bool) MarshalOption {
	return func(m marshalOptions) (marshalOptions, error) {
		m.UseEnumNumbers = use
		return m, nil
	}
}

var marshalPool = &marshallerPool{
	pool: sync.NewPool[*bytes.Buffer](
		context.Background(),
		"clawjson.marshallerPool",
		func() *bytes.Buffer {
			b := &bytes.Buffer{}
			b.Grow(256)
			return b
		},
	),
}

// marshalState holds reusable state for marshaling to avoid allocations.
type marshalState struct {
	firstStack []bool
	scratch    []byte
}

var marshalStatePool = sync.NewPool[*marshalState](
	context.Background(),
	"clawjson.marshalStatePool",
	func() *marshalState {
		return &marshalState{
			firstStack: make([]bool, 0, 8),
			scratch:    make([]byte, 0, 64),
		}
	},
)

func getMarshalState(ctx context.Context) *marshalState {
	s := marshalStatePool.Get(ctx)
	s.firstStack = s.firstStack[:0]
	s.scratch = s.scratch[:0]
	return s
}

func putMarshalState(ctx context.Context, s *marshalState) {
	marshalStatePool.Put(ctx, s)
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

	m.pool.Put(ctx, b)
}

// Marshal marshals the Walkable to JSON.
func Marshal(ctx context.Context, v Walkable, options ...MarshalOption) (Buffer, error) {
	buf := marshalPool.get(ctx)
	if err := MarshalWriter(ctx, v, buf, options...); err != nil {
		return Buffer{}, err
	}
	return Buffer{buf}, nil
}

// MarshalWriter marshals the Walkable to JSON, writing to the provided io.Writer.
func MarshalWriter(ctx context.Context, v Walkable, w io.Writer, options ...MarshalOption) error {
	opts := marshalOptions{}
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return err
		}
	}
	return writeJSON(ctx, w, v, opts)
}

// writeJSON writes JSON from the token stream to an io.Writer.
func writeJSON(ctx context.Context, w io.Writer, walker Walkable, opts marshalOptions) error {
	// Get pooled state to avoid allocations
	state := getMarshalState(ctx)
	defer putMarshalState(ctx, state)

	var writeErr error
	walker.Walk(ctx, func(tok clawiter.Token) bool {
		switch tok.Kind {
		case clawiter.TokenStructStart:
			if _, err := w.Write(jsonOpenBrace); err != nil {
				writeErr = err
				return false
			}
			state.firstStack = append(state.firstStack, true)

		case clawiter.TokenStructEnd:
			if _, err := w.Write(jsonCloseBrace); err != nil {
				writeErr = err
				return false
			}
			if len(state.firstStack) > 0 {
				state.firstStack = state.firstStack[:len(state.firstStack)-1]
			}

		case clawiter.TokenListStart:
			if _, err := w.Write(jsonOpenBracket); err != nil {
				writeErr = err
				return false
			}
			state.firstStack = append(state.firstStack, true)

		case clawiter.TokenListEnd:
			if _, err := w.Write(jsonCloseBracket); err != nil {
				writeErr = err
				return false
			}
			if len(state.firstStack) > 0 {
				state.firstStack = state.firstStack[:len(state.firstStack)-1]
			}

		case clawiter.TokenField:
			// Write comma if not first element
			if len(state.firstStack) > 0 {
				if !state.firstStack[len(state.firstStack)-1] {
					if _, err := w.Write(jsonComma); err != nil {
						writeErr = err
						return false
					}
				}
				state.firstStack[len(state.firstStack)-1] = false
			}

			// Write field name if present (not present for list items)
			if tok.Name != "" {
				state.scratch = appendEscapedString(state.scratch[:0], tok.Name)
				if _, err := w.Write(state.scratch); err != nil {
					writeErr = err
					return false
				}
				if _, err := w.Write(jsonColon); err != nil {
					writeErr = err
					return false
				}
			}

			// Write field value based on type
			if err := writeValue(w, tok, opts, &state.scratch); err != nil {
				writeErr = err
				return false
			}
		}
		return true
	})
	return writeErr
}

// writeValue writes the JSON value for a field token.
// scratch is a reusable buffer for formatting values.
func writeValue(w io.Writer, tok clawiter.Token, opts marshalOptions, scratch *[]byte) error {
	// Handle nil structs and lists
	if tok.IsNil {
		_, err := w.Write(jsonNull)
		return err
	}

	// Handle struct and list announcements (values come from nested tokens)
	switch tok.Type {
	case field.FTStruct, field.FTListStructs,
		field.FTListBools, field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
		field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
		field.FTListFloat32, field.FTListFloat64, field.FTListBytes, field.FTListStrings:
		return nil // Value handled by nested tokens (ListStart, items, ListEnd)
	}

	// Handle enums (individual values, not list announcements)
	if tok.IsEnum {
		if opts.UseEnumNumbers {
			switch tok.Type {
			case field.FTUint8:
				*scratch = strconv.AppendUint((*scratch)[:0], uint64(tok.Uint8()), 10)
				_, err := w.Write(*scratch)
				return err
			case field.FTUint16:
				*scratch = strconv.AppendUint((*scratch)[:0], uint64(tok.Uint16()), 10)
				_, err := w.Write(*scratch)
				return err
			}
		}
		*scratch = appendEscapedString((*scratch)[:0], tok.EnumName)
		_, err := w.Write(*scratch)
		return err
	}

	// Handle scalar types
	switch tok.Type {
	case field.FTBool:
		if tok.Bool() {
			_, err := w.Write(jsonTrue)
			return err
		}
		_, err := w.Write(jsonFalse)
		return err

	case field.FTInt8:
		*scratch = strconv.AppendInt((*scratch)[:0], int64(tok.Int8()), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTInt16:
		*scratch = strconv.AppendInt((*scratch)[:0], int64(tok.Int16()), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTInt32:
		*scratch = strconv.AppendInt((*scratch)[:0], int64(tok.Int32()), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTInt64:
		*scratch = strconv.AppendInt((*scratch)[:0], tok.Int64(), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTUint8:
		*scratch = strconv.AppendUint((*scratch)[:0], uint64(tok.Uint8()), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTUint16:
		*scratch = strconv.AppendUint((*scratch)[:0], uint64(tok.Uint16()), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTUint32:
		*scratch = strconv.AppendUint((*scratch)[:0], uint64(tok.Uint32()), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTUint64:
		*scratch = strconv.AppendUint((*scratch)[:0], tok.Uint64(), 10)
		_, err := w.Write(*scratch)
		return err

	case field.FTFloat32:
		f := tok.Float32()
		if math.IsInf(float64(f), 0) || math.IsNaN(float64(f)) {
			_, err := w.Write(jsonNull)
			return err
		}
		*scratch = strconv.AppendFloat((*scratch)[:0], float64(f), 'g', -1, 32)
		_, err := w.Write(*scratch)
		return err

	case field.FTFloat64:
		f := tok.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			_, err := w.Write(jsonNull)
			return err
		}
		*scratch = strconv.AppendFloat((*scratch)[:0], f, 'g', -1, 64)
		_, err := w.Write(*scratch)
		return err

	case field.FTString:
		*scratch = appendEscapedString((*scratch)[:0], tok.String())
		_, err := w.Write(*scratch)
		return err

	case field.FTBytes:
		// For bytes, base64 encode directly into scratch buffer with quotes
		// This avoids allocating an intermediate string
		*scratch = append((*scratch)[:0], '"')
		*scratch = base64.StdEncoding.AppendEncode(*scratch, tok.Bytes)
		*scratch = append(*scratch, '"')
		_, err := w.Write(*scratch)
		return err
	}

	return fmt.Errorf("unsupported field type: %v", tok.Type)
}

// appendEscapedString appends a JSON-escaped string (with quotes) to dst.
// This is an optimized version that avoids json.Marshal for common cases.
func appendEscapedString(dst []byte, s string) []byte {
	dst = append(dst, '"')

	// Get unsafe byte view of string to avoid allocation during scan
	sb := stringToBytes(s)

	// Fast path: check if the string needs escaping
	needsEscape := false
	for _, c := range sb {
		if c < 0x20 || c == '"' || c == '\\' || c > 0x7e {
			needsEscape = true
			break
		}
	}

	if !needsEscape {
		// Fast path: no escaping needed, append directly from unsafe view
		dst = append(dst, sb...)
	} else {
		// Slow path: escape special characters
		for _, r := range s {
			switch {
			case r == '"':
				dst = append(dst, '\\', '"')
			case r == '\\':
				dst = append(dst, '\\', '\\')
			case r == '\n':
				dst = append(dst, '\\', 'n')
			case r == '\r':
				dst = append(dst, '\\', 'r')
			case r == '\t':
				dst = append(dst, '\\', 't')
			case r < 0x20:
				// Control characters: use \uXXXX
				dst = append(dst, '\\', 'u', '0', '0')
				dst = append(dst, hexDigits[r>>4], hexDigits[r&0xf])
			case r > utf8.MaxRune:
				// Invalid rune
				dst = append(dst, '\\', 'u', 'f', 'f', 'f', 'd')
			default:
				// Valid UTF-8, just append
				dst = utf8.AppendRune(dst, r)
			}
		}
	}

	dst = append(dst, '"')
	return dst
}

const hexDigits = "0123456789abcdef"

// stringToBytes converts a string to a byte slice without copying.
// WARNING: The returned slice must NOT be modified.
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Array is used to write out an array of JSON objects.
type Array struct {
	writer  io.Writer
	opts    marshalOptions
	written bool
}

// NewArray creates a new Array for streaming JSON array output.
func NewArray(w io.Writer, options ...MarshalOption) (*Array, error) {
	opts := marshalOptions{}
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return nil, err
		}
	}
	return &Array{writer: w, opts: opts}, nil
}

// Write writes a Walkable to the JSON array.
func (a *Array) Write(ctx context.Context, v Walkable, options ...MarshalOption) error {
	opts := a.opts
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return err
		}
	}

	if !a.written {
		if _, err := a.writer.Write(jsonOpenBracket); err != nil {
			return err
		}
		a.written = true
	} else {
		if _, err := a.writer.Write(jsonComma); err != nil {
			return err
		}
	}

	return writeJSON(ctx, a.writer, v, opts)
}

// Close finishes writing the JSON array.
func (a *Array) Close() error {
	if !a.written {
		_, err := a.writer.Write(jsonEmptyArray)
		return err
	}
	_, err := a.writer.Write(jsonCloseBracket)
	return err
}

// Reset resets the Array to write to a new io.Writer.
func (a *Array) Reset(w io.Writer) {
	a.written = false
	a.writer = w
}

// Ingester is an interface for ingesting Claw tokens into a struct.
type Ingester interface {
	Ingest(context.Context, clawiter.Walker, ...clawiter.IngestOption) error
}

// unmarshalOptions provides options for reading JSON into Claw structs.
type unmarshalOptions struct {
	IgnoreUnknownFields bool
}

// UnmarshalOption provides options for unmarshaling JSON to Claw.
type UnmarshalOption func(unmarshalOptions) (unmarshalOptions, error)

// WithIgnoreUnknownFields configures whether unknown JSON fields should be ignored.
func WithIgnoreUnknownFields(ignore bool) UnmarshalOption {
	return func(u unmarshalOptions) (unmarshalOptions, error) {
		u.IgnoreUnknownFields = ignore
		return u, nil
	}
}

// Unmarshal parses JSON data and populates the Ingester.
func Unmarshal(ctx context.Context, data []byte, v Ingester, options ...UnmarshalOption) error {
	return UnmarshalReader(ctx, bytes.NewReader(data), v, options...)
}

// UnmarshalReader parses JSON from a reader and populates the Ingester.
func UnmarshalReader(ctx context.Context, r io.Reader, v Ingester, options ...UnmarshalOption) error {
	opts := unmarshalOptions{}
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return err
		}
	}

	tokens := jsonToTokens(r)
	// Convert iter.Seq to clawiter.Walker (both are func(func(Token) bool) but need explicit conversion)
	walker := clawiter.Walker(func(yield clawiter.YieldToken) {
		tokens(yield)
	})
	var ingestOpts []clawiter.IngestOption
	if opts.IgnoreUnknownFields {
		ingestOpts = append(ingestOpts, clawiter.WithIgnoreUnknownFields(true))
	}
	return v.Ingest(ctx, walker, ingestOpts...)
}

// jsonToTokens parses JSON and yields Claw tokens.
func jsonToTokens(r io.Reader) iter.Seq[clawiter.Token] {
	return func(yield func(clawiter.Token) bool) {
		dec := jsontext.NewDecoder(r)

		// Stack to track context at each nesting level
		// true = in object (expecting key/value pairs), false = in array
		var contextStack []bool
		// Stack to track if we're expecting a key (true) or value (false) in objects
		var expectKeyStack []bool
		// Current field name for the next value
		var currentName string

		inObject := func() bool {
			if len(contextStack) == 0 {
				return false
			}
			return contextStack[len(contextStack)-1]
		}

		expectKey := func() bool {
			if len(expectKeyStack) == 0 {
				return false
			}
			return expectKeyStack[len(expectKeyStack)-1]
		}

		setExpectKey := func(v bool) {
			if len(expectKeyStack) > 0 {
				expectKeyStack[len(expectKeyStack)-1] = v
			}
		}

		for {
			tok, err := dec.ReadToken()
			if err != nil {
				if err == io.EOF {
					return
				}
				return
			}

			switch tok.Kind() {
			case '{':
				// Object start
				// If we have a current field name, emit a TokenField first (for nested structs)
				if currentName != "" {
					if !yield(clawiter.Token{Kind: clawiter.TokenField, Name: currentName, Type: field.FTStruct}) {
						return
					}
				}
				// Emit TokenStructStart
				structName := currentName
				if structName == "" {
					structName = "root"
				}
				if !yield(clawiter.Token{Kind: clawiter.TokenStructStart, Name: structName}) {
					return
				}
				// Push object context - expecting keys next
				contextStack = append(contextStack, true)
				expectKeyStack = append(expectKeyStack, true)
				currentName = ""

			case '}':
				// Object end - emit TokenStructEnd
				if !yield(clawiter.Token{Kind: clawiter.TokenStructEnd}) {
					return
				}
				// Pop context
				if len(contextStack) > 0 {
					contextStack = contextStack[:len(contextStack)-1]
				}
				if len(expectKeyStack) > 0 {
					expectKeyStack = expectKeyStack[:len(expectKeyStack)-1]
				}
				// After closing an object, the parent expects next key (if in object)
				setExpectKey(true)

			case '[':
				// Array start
				// If we have a current field name, emit a TokenField first (for lists)
				if currentName != "" {
					if !yield(clawiter.Token{Kind: clawiter.TokenField, Name: currentName}) {
						return
					}
				}
				// Emit TokenListStart
				if !yield(clawiter.Token{Kind: clawiter.TokenListStart}) {
					return
				}
				// Push array context
				contextStack = append(contextStack, false)
				expectKeyStack = append(expectKeyStack, false)
				currentName = ""

			case ']':
				// Array end - emit TokenListEnd
				if !yield(clawiter.Token{Kind: clawiter.TokenListEnd}) {
					return
				}
				// Pop context
				if len(contextStack) > 0 {
					contextStack = contextStack[:len(contextStack)-1]
				}
				if len(expectKeyStack) > 0 {
					expectKeyStack = expectKeyStack[:len(expectKeyStack)-1]
				}
				// After closing an array, the parent expects next key (if in object)
				setExpectKey(true)

			case '"':
				// String - could be a field name or a value
				s := tok.String()

				// If we're in an object and expecting a key, this is a field name
				if inObject() && expectKey() {
					currentName = s
					setExpectKey(false) // Next we expect a value
					continue
				}

				// This is a string value - emit as field token
				token := clawiter.Token{
					Kind:  clawiter.TokenField,
					Name:  currentName,
					Type:  field.FTString,
					Bytes: []byte(s),
				}
				if !yield(token) {
					return
				}
				currentName = ""
				setExpectKey(true) // Next we expect a key (if in object)

			case '0':
				// Number - store as int64 (will be cast by Ingest as needed)
				token := clawiter.Token{
					Kind: clawiter.TokenField,
					Name: currentName,
				}
				// Use Int64 which handles most JSON numbers correctly
				// Ingest will cast to the appropriate type (uint8, uint16, int32, etc.)
				token.SetInt64(tok.Int())
				if !yield(token) {
					return
				}
				currentName = ""
				setExpectKey(true)

			case 't', 'f':
				// Boolean
				b := tok.Bool()
				token := clawiter.Token{
					Kind: clawiter.TokenField,
					Name: currentName,
					Type: field.FTBool,
				}
				token.SetBool(b)
				if !yield(token) {
					return
				}
				currentName = ""
				setExpectKey(true)

			case 'n':
				// Null
				token := clawiter.Token{
					Kind:  clawiter.TokenField,
					Name:  currentName,
					IsNil: true,
				}
				if !yield(token) {
					return
				}
				currentName = ""
				setExpectKey(true)
			}
		}
	}
}

// Ensure imports are used
var _ = jsonv2.Marshal
