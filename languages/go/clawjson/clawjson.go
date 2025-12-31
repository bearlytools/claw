// Package clawjson provides functionality to marshal Claw structures to JSON.
package clawjson

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"math"
	"strconv"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	jsonv2 "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Walker is an interface for walking over Claw tokens.
type Walker interface {
	Walk(ctx context.Context) iter.Seq[clawiter.Token]
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

// Marshal marshals the Walker to JSON.
func Marshal(ctx context.Context, v Walker, options ...MarshalOption) ([]byte, error) {
	var buf bytes.Buffer
	if err := MarshalWriter(ctx, v, &buf, options...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalWriter marshals the Walker to JSON, writing to the provided io.Writer.
func MarshalWriter(ctx context.Context, v Walker, w io.Writer, options ...MarshalOption) error {
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
func writeJSON(ctx context.Context, w io.Writer, walker Walker, opts marshalOptions) error {
	// Stack to track whether we need commas (true = first element, no comma needed)
	// Each entry represents a nesting level (struct or list)
	firstStack := []bool{}

	for tok := range walker.Walk(ctx) {
		switch tok.Kind {
		case clawiter.TokenStructStart:
			if _, err := w.Write([]byte("{")); err != nil {
				return err
			}
			firstStack = append(firstStack, true)

		case clawiter.TokenStructEnd:
			if _, err := w.Write([]byte("}")); err != nil {
				return err
			}
			if len(firstStack) > 0 {
				firstStack = firstStack[:len(firstStack)-1]
			}

		case clawiter.TokenListStart:
			if _, err := w.Write([]byte("[")); err != nil {
				return err
			}
			firstStack = append(firstStack, true)

		case clawiter.TokenListEnd:
			if _, err := w.Write([]byte("]")); err != nil {
				return err
			}
			if len(firstStack) > 0 {
				firstStack = firstStack[:len(firstStack)-1]
			}

		case clawiter.TokenField:
			// Write comma if not first element
			if len(firstStack) > 0 {
				if !firstStack[len(firstStack)-1] {
					if _, err := w.Write([]byte(",")); err != nil {
						return err
					}
				}
				firstStack[len(firstStack)-1] = false
			}

			// Write field name if present (not present for list items)
			if tok.Name != "" {
				escaped, err := json.Marshal(tok.Name)
				if err != nil {
					return fmt.Errorf("failed to escape field name %q: %w", tok.Name, err)
				}
				if _, err := w.Write(escaped); err != nil {
					return err
				}
				if _, err := w.Write([]byte(":")); err != nil {
					return err
				}
			}

			// Write field value based on type
			if err := writeValue(w, tok, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeValue writes the JSON value for a field token.
func writeValue(w io.Writer, tok clawiter.Token, opts marshalOptions) error {
	// Handle nil structs and lists
	if tok.IsNil {
		_, err := w.Write([]byte("null"))
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
				_, err := w.Write([]byte(strconv.FormatUint(uint64(tok.Uint8()), 10)))
				return err
			case field.FTUint16:
				_, err := w.Write([]byte(strconv.FormatUint(uint64(tok.Uint16()), 10)))
				return err
			}
		}
		escaped, err := json.Marshal(tok.EnumName)
		if err != nil {
			return fmt.Errorf("failed to escape enum name %q: %w", tok.EnumName, err)
		}
		_, err = w.Write(escaped)
		return err
	}

	// Handle scalar types
	switch tok.Type {
	case field.FTBool:
		if tok.Bool() {
			_, err := w.Write([]byte("true"))
			return err
		}
		_, err := w.Write([]byte("false"))
		return err

	case field.FTInt8:
		_, err := w.Write([]byte(strconv.FormatInt(int64(tok.Int8()), 10)))
		return err

	case field.FTInt16:
		_, err := w.Write([]byte(strconv.FormatInt(int64(tok.Int16()), 10)))
		return err

	case field.FTInt32:
		_, err := w.Write([]byte(strconv.FormatInt(int64(tok.Int32()), 10)))
		return err

	case field.FTInt64:
		_, err := w.Write([]byte(strconv.FormatInt(tok.Int64(), 10)))
		return err

	case field.FTUint8:
		_, err := w.Write([]byte(strconv.FormatUint(uint64(tok.Uint8()), 10)))
		return err

	case field.FTUint16:
		_, err := w.Write([]byte(strconv.FormatUint(uint64(tok.Uint16()), 10)))
		return err

	case field.FTUint32:
		_, err := w.Write([]byte(strconv.FormatUint(uint64(tok.Uint32()), 10)))
		return err

	case field.FTUint64:
		_, err := w.Write([]byte(strconv.FormatUint(tok.Uint64(), 10)))
		return err

	case field.FTFloat32:
		f := tok.Float32()
		if math.IsInf(float64(f), 0) || math.IsNaN(float64(f)) {
			_, err := w.Write([]byte("null"))
			return err
		}
		_, err := w.Write([]byte(strconv.FormatFloat(float64(f), 'g', -1, 32)))
		return err

	case field.FTFloat64:
		f := tok.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			_, err := w.Write([]byte("null"))
			return err
		}
		_, err := w.Write([]byte(strconv.FormatFloat(f, 'g', -1, 64)))
		return err

	case field.FTString:
		escaped, err := json.Marshal(tok.String())
		if err != nil {
			return fmt.Errorf("failed to escape string: %w", err)
		}
		_, err = w.Write(escaped)
		return err

	case field.FTBytes:
		encoded := base64.StdEncoding.EncodeToString(tok.Bytes)
		escaped, err := json.Marshal(encoded)
		if err != nil {
			return fmt.Errorf("failed to escape bytes: %w", err)
		}
		_, err = w.Write(escaped)
		return err
	}

	return fmt.Errorf("unsupported field type: %v", tok.Type)
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

// Write writes a Walker to the JSON array.
func (a *Array) Write(ctx context.Context, v Walker, options ...MarshalOption) error {
	opts := a.opts
	for _, opt := range options {
		var err error
		opts, err = opt(opts)
		if err != nil {
			return err
		}
	}

	if !a.written {
		if _, err := a.writer.Write([]byte("[")); err != nil {
			return err
		}
		a.written = true
	} else {
		if _, err := a.writer.Write([]byte(",")); err != nil {
			return err
		}
	}

	return writeJSON(ctx, a.writer, v, opts)
}

// Close finishes writing the JSON array.
func (a *Array) Close() error {
	if !a.written {
		_, err := a.writer.Write([]byte("[]"))
		return err
	}
	_, err := a.writer.Write([]byte("]"))
	return err
}

// Reset resets the Array to write to a new io.Writer.
func (a *Array) Reset(w io.Writer) {
	a.written = false
	a.writer = w
}

// Ingester is an interface for ingesting Claw tokens into a struct.
type Ingester interface {
	IngestWithOptions(context.Context, iter.Seq[clawiter.Token], clawiter.IngestOptions) error
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
	ingestOpts := clawiter.IngestOptions{
		IgnoreUnknownFields: opts.IgnoreUnknownFields,
	}
	return v.IngestWithOptions(ctx, tokens, ingestOpts)
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
