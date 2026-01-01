package clawtext

import (
	"encoding/base64"
	"io"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/concurrency/sync"
)

// Pre-allocated byte slices for common clawtext tokens.
var (
	textOpenBrace    = []byte("{")
	textCloseBrace   = []byte("}")
	textOpenBracket  = []byte("[")
	textCloseBracket = []byte("]")
	textComma        = []byte(",")
	textColon        = []byte(": ")
	textNull         = []byte("null")
	textTrue         = []byte("true")
	textFalse        = []byte("false")
	textNewline      = []byte("\n")
	textAtMap        = []byte("@map ")
	textIndent       = []byte("    ") // 4 spaces default
)

// Walkable is an interface for types that can walk over Claw tokens.
type Walkable interface {
	Walk(ctx context.Context, yield clawiter.YieldToken, opts ...clawiter.WalkOption)
}

// marshalOptions provides options for writing Claw output to clawtext.
type marshalOptions struct {
	UseEnumNumbers bool
	UseHexBytes    bool
	Indent         string
}

// MarshalOption provides options for marshaling Claw to clawtext.
type MarshalOption func(marshalOptions) (marshalOptions, error)

// WithUseEnumNumbers configures whether enum values are emitted as numbers or strings.
func WithUseEnumNumbers(use bool) MarshalOption {
	return func(m marshalOptions) (marshalOptions, error) {
		m.UseEnumNumbers = use
		return m, nil
	}
}

// WithUseHexBytes configures whether byte arrays are emitted as hex instead of base64.
func WithUseHexBytes(use bool) MarshalOption {
	return func(m marshalOptions) (marshalOptions, error) {
		m.UseHexBytes = use
		return m, nil
	}
}

// WithIndent configures the indentation string (default is 4 spaces).
func WithIndent(indent string) MarshalOption {
	return func(m marshalOptions) (marshalOptions, error) {
		m.Indent = indent
		return m, nil
	}
}

// marshalState holds reusable state for marshaling.
type marshalState struct {
	depthStack   []bool   // track first element at each depth
	depth        int      // current indentation depth
	scratch      []byte   // reusable buffer
	isTopLevel   bool     // true for root struct (no braces)
	inArray      bool     // true when inside an array
	inMap        bool     // true when inside a map
	pendingComma bool     // need to write comma before next element
}

var marshalStatePool = sync.NewPool[*marshalState](
	context.Background(),
	"clawtext.marshalStatePool",
	func() *marshalState {
		return &marshalState{
			depthStack: make([]bool, 0, 8),
			scratch:    make([]byte, 0, 64),
		}
	},
)

func getMarshalState(ctx context.Context) *marshalState {
	s := marshalStatePool.Get(ctx)
	s.depthStack = s.depthStack[:0]
	s.scratch = s.scratch[:0]
	s.depth = 0
	s.isTopLevel = true
	s.inArray = false
	s.inMap = false
	s.pendingComma = false
	return s
}

func putMarshalState(ctx context.Context, s *marshalState) {
	marshalStatePool.Put(ctx, s)
}

// writeClawtext writes clawtext from the token stream to an io.Writer.
func writeClawtext(ctx context.Context, w io.Writer, walker Walkable, opts marshalOptions) error {
	state := getMarshalState(ctx)
	defer putMarshalState(ctx, state)

	indent := textIndent
	if opts.Indent != "" {
		indent = []byte(opts.Indent)
	}

	var writeErr error
	walker.Walk(ctx, func(tok clawiter.Token) bool {
		switch tok.Kind {
		case clawiter.TokenStructStart:
			if state.isTopLevel {
				// Root struct - don't write braces
				state.isTopLevel = false
				state.depthStack = append(state.depthStack, true)
			} else {
				// Nested struct - write opening brace
				if _, err := w.Write(textOpenBrace); err != nil {
					writeErr = err
					return false
				}
				if _, err := w.Write(textNewline); err != nil {
					writeErr = err
					return false
				}
				state.depth++
				state.depthStack = append(state.depthStack, true)
			}

		case clawiter.TokenStructEnd:
			if len(state.depthStack) > 0 {
				state.depthStack = state.depthStack[:len(state.depthStack)-1]
			}
			if state.depth > 0 {
				state.depth--
				// Write closing brace with proper indentation
				if err := writeIndent(w, indent, state.depth); err != nil {
					writeErr = err
					return false
				}
				if _, err := w.Write(textCloseBrace); err != nil {
					writeErr = err
					return false
				}
				// Write comma after nested struct
				if _, err := w.Write(textComma); err != nil {
					writeErr = err
					return false
				}
				if _, err := w.Write(textNewline); err != nil {
					writeErr = err
					return false
				}
			}

		case clawiter.TokenListStart:
			if _, err := w.Write(textOpenBracket); err != nil {
				writeErr = err
				return false
			}
			state.depthStack = append(state.depthStack, true)
			state.inArray = true

		case clawiter.TokenListEnd:
			if len(state.depthStack) > 0 {
				state.depthStack = state.depthStack[:len(state.depthStack)-1]
			}
			state.inArray = false
			if _, err := w.Write(textCloseBracket); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textComma); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textNewline); err != nil {
				writeErr = err
				return false
			}

		case clawiter.TokenMapStart:
			// Write @map { for maps
			if _, err := w.Write(textAtMap); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textOpenBrace); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textNewline); err != nil {
				writeErr = err
				return false
			}
			state.depth++
			state.depthStack = append(state.depthStack, true)
			state.inMap = true

		case clawiter.TokenMapEnd:
			if len(state.depthStack) > 0 {
				state.depthStack = state.depthStack[:len(state.depthStack)-1]
			}
			state.depth--
			state.inMap = false
			if err := writeIndent(w, indent, state.depth); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textCloseBrace); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textComma); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textNewline); err != nil {
				writeErr = err
				return false
			}

		case clawiter.TokenMapEntry:
			// Write comma before non-first entries
			if len(state.depthStack) > 0 && !state.depthStack[len(state.depthStack)-1] {
				// Already written on previous line
			}
			if len(state.depthStack) > 0 {
				state.depthStack[len(state.depthStack)-1] = false
			}

			// Write key
			if err := writeIndent(w, indent, state.depth); err != nil {
				writeErr = err
				return false
			}
			state.scratch = appendEscapedString(state.scratch[:0], tok.KeyString())
			if _, err := w.Write(state.scratch); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textColon); err != nil {
				writeErr = err
				return false
			}

			// Write value
			if err := writeValue(w, tok, opts, &state.scratch); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textComma); err != nil {
				writeErr = err
				return false
			}
			if _, err := w.Write(textNewline); err != nil {
				writeErr = err
				return false
			}

		case clawiter.TokenField:
			// Write comma before non-first fields (in arrays only for inline values)
			if state.inArray && len(state.depthStack) > 0 && !state.depthStack[len(state.depthStack)-1] {
				if _, err := w.Write([]byte(", ")); err != nil {
					writeErr = err
					return false
				}
			}
			if len(state.depthStack) > 0 {
				state.depthStack[len(state.depthStack)-1] = false
			}

			// Write field name if present (not present for list items)
			if tok.Name != "" {
				if err := writeIndent(w, indent, state.depth); err != nil {
					writeErr = err
					return false
				}
				if _, err := w.Write([]byte(tok.Name)); err != nil {
					writeErr = err
					return false
				}
				if _, err := w.Write(textColon); err != nil {
					writeErr = err
					return false
				}
			}

			// Write field value based on type
			// For structs and lists, the value is written by subsequent tokens,
			// so don't write comma/newline here - they handle their own
			switch tok.Type {
			case field.FTStruct:
				// Struct value handled by TokenStructStart/End
				if tok.IsNil {
					if _, err := w.Write(textNull); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textComma); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textNewline); err != nil {
						writeErr = err
						return false
					}
				}
				// Non-nil struct - value comes from nested tokens
			case field.FTListBools, field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
				field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
				field.FTListFloat32, field.FTListFloat64, field.FTListBytes, field.FTListStrings, field.FTListStructs:
				// List value handled by TokenListStart/End
				if tok.IsNil {
					if _, err := w.Write(textNull); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textComma); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textNewline); err != nil {
						writeErr = err
						return false
					}
				}
				// Non-nil list - value comes from nested tokens
			case field.FTMap:
				// Map value handled by TokenMapStart/End
				if tok.IsNil {
					if _, err := w.Write(textNull); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textComma); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textNewline); err != nil {
						writeErr = err
						return false
					}
				}
				// Non-nil map - value comes from nested tokens
			default:
				// Scalar value - write it directly
				if err := writeValue(w, tok, opts, &state.scratch); err != nil {
					writeErr = err
					return false
				}

				// Write comma and newline for non-array items
				if !state.inArray {
					if _, err := w.Write(textComma); err != nil {
						writeErr = err
						return false
					}
					if _, err := w.Write(textNewline); err != nil {
						writeErr = err
						return false
					}
				}
			}
		}
		return true
	})
	return writeErr
}

// writeIndent writes the appropriate indentation.
func writeIndent(w io.Writer, indent []byte, depth int) error {
	for i := 0; i < depth; i++ {
		if _, err := w.Write(indent); err != nil {
			return err
		}
	}
	return nil
}

// writeValue writes the clawtext value for a field token.
func writeValue(w io.Writer, tok clawiter.Token, opts marshalOptions, scratch *[]byte) error {
	// Handle nil structs and lists
	if tok.IsNil {
		_, err := w.Write(textNull)
		return err
	}

	// Handle struct and list announcements (values come from nested tokens)
	switch tok.Type {
	case field.FTStruct, field.FTListStructs,
		field.FTListBools, field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
		field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
		field.FTListFloat32, field.FTListFloat64, field.FTListBytes, field.FTListStrings,
		field.FTMap:
		return nil // Value handled by nested tokens
	}

	// Handle enums
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
		// Write enum name without quotes
		_, err := w.Write([]byte(tok.EnumName))
		return err
	}

	// Handle scalar types
	switch tok.Type {
	case field.FTBool:
		if tok.Bool() {
			_, err := w.Write(textTrue)
			return err
		}
		_, err := w.Write(textFalse)
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
			_, err := w.Write(textNull)
			return err
		}
		*scratch = strconv.AppendFloat((*scratch)[:0], float64(f), 'g', -1, 32)
		_, err := w.Write(*scratch)
		return err

	case field.FTFloat64:
		f := tok.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			_, err := w.Write(textNull)
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
		if opts.UseHexBytes {
			// Write as 0x prefixed hex
			*scratch = append((*scratch)[:0], '0', 'x')
			*scratch = appendHex(*scratch, tok.Bytes)
			_, err := w.Write(*scratch)
			return err
		}
		// Write as base64 quoted string
		*scratch = append((*scratch)[:0], '"')
		*scratch = base64.StdEncoding.AppendEncode(*scratch, tok.Bytes)
		*scratch = append(*scratch, '"')
		_, err := w.Write(*scratch)
		return err
	}

	return nil
}

const hexDigits = "0123456789abcdef"

// appendHex appends the hex encoding of data to dst.
func appendHex(dst []byte, data []byte) []byte {
	for _, b := range data {
		dst = append(dst, hexDigits[b>>4], hexDigits[b&0x0f])
	}
	return dst
}

// appendEscapedString appends a clawtext-escaped string (with quotes) to dst.
func appendEscapedString(dst []byte, s string) []byte {
	dst = append(dst, '"')

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

	dst = append(dst, '"')
	return dst
}
