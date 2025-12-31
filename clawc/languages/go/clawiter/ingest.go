// Package clawiter provides types for streaming iteration over Claw structs.
package clawiter

import (
	"fmt"
	"iter"

	"github.com/bearlytools/claw/clawc/languages/go/field"
)

// IngestOptions holds configuration for Ingest.
// This is populated by IngestOption functions and passed to XXXIngestFrom.
type IngestOptions struct {
	// IgnoreUnknownFields causes unknown field names to be skipped instead of returning an error.
	IgnoreUnknownFields bool
}

// IngestOption configures Ingest behavior.
type IngestOption func(IngestOptions) (IngestOptions, error)

// WithIgnoreUnknownFields sets whether to ignore unknown fields during ingestion.
// When true, unknown fields are skipped instead of causing an error.
func WithIgnoreUnknownFields(ignore bool) IngestOption {
	return func(o IngestOptions) (IngestOptions, error) {
		o.IgnoreUnknownFields = ignore
		return o, nil
	}
}

// TokenStream wraps an iter.Seq[Token] for pull-based consumption with peek support.
type TokenStream struct {
	next   func() (Token, bool)
	stop   func()
	peeked *Token
}

// NewTokenStream creates a TokenStream from an iter.Seq[Token].
func NewTokenStream(seq iter.Seq[Token]) *TokenStream {
	next, stop := iter.Pull(seq)
	return &TokenStream{next: next, stop: stop}
}

// Next returns the next token from the stream.
func (ts *TokenStream) Next() (Token, bool) {
	if ts.peeked != nil {
		tok := *ts.peeked
		ts.peeked = nil
		return tok, true
	}
	return ts.next()
}

// Peek returns the next token without consuming it.
func (ts *TokenStream) Peek() (Token, bool) {
	if ts.peeked != nil {
		return *ts.peeked, true
	}
	tok, ok := ts.next()
	if ok {
		ts.peeked = &tok
	}
	return tok, ok
}

// Close releases resources associated with the token stream.
func (ts *TokenStream) Close() {
	ts.stop()
}

// SkipValue skips a field value in the token stream, including nested structs and lists.
func SkipValue(ts *TokenStream, fieldTok Token) error {
	switch fieldTok.Type {
	case field.FTStruct:
		if fieldTok.IsNil {
			return nil
		}
		return skipStruct(ts)
	case field.FTListStructs:
		if fieldTok.IsNil {
			return nil
		}
		return skipListStructs(ts)
	case field.FTListBools, field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
		field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
		field.FTListFloat32, field.FTListFloat64, field.FTListBytes, field.FTListStrings:
		if fieldTok.IsNil {
			return nil
		}
		return skipList(ts)
	case field.FTMap:
		if fieldTok.IsNil {
			return nil
		}
		return skipMap(ts, fieldTok.ValueType)
	default:
		// Scalar values are already consumed in the field token
		return nil
	}
}

// skipStruct consumes and discards a struct's tokens (StructStart through StructEnd).
func skipStruct(ts *TokenStream) error {
	tok, ok := ts.Next()
	if !ok {
		return fmt.Errorf("expected TokenStructStart, got EOF")
	}
	if tok.Kind != TokenStructStart {
		return fmt.Errorf("expected TokenStructStart, got %v", tok.Kind)
	}

	depth := 1
	for depth > 0 {
		tok, ok = ts.Next()
		if !ok {
			return fmt.Errorf("unexpected EOF while skipping struct")
		}
		switch tok.Kind {
		case TokenStructStart:
			depth++
		case TokenStructEnd:
			depth--
		}
	}
	return nil
}

// skipList consumes and discards a list's tokens (ListStart through ListEnd).
func skipList(ts *TokenStream) error {
	tok, ok := ts.Next()
	if !ok {
		return fmt.Errorf("expected TokenListStart, got EOF")
	}
	if tok.Kind != TokenListStart {
		return fmt.Errorf("expected TokenListStart, got %v", tok.Kind)
	}

	for {
		tok, ok = ts.Next()
		if !ok {
			return fmt.Errorf("unexpected EOF while skipping list")
		}
		if tok.Kind == TokenListEnd {
			return nil
		}
		// Skip scalar field tokens (they don't have nested content)
	}
}

// skipListStructs consumes and discards a list of structs (ListStart, structs, ListEnd).
func skipListStructs(ts *TokenStream) error {
	tok, ok := ts.Next()
	if !ok {
		return fmt.Errorf("expected TokenListStart, got EOF")
	}
	if tok.Kind != TokenListStart {
		return fmt.Errorf("expected TokenListStart, got %v", tok.Kind)
	}

	for {
		peekTok, ok := ts.Peek()
		if !ok {
			return fmt.Errorf("unexpected EOF while skipping list of structs")
		}
		if peekTok.Kind == TokenListEnd {
			ts.Next() // consume the ListEnd
			return nil
		}
		// Skip each struct in the list
		if err := skipStruct(ts); err != nil {
			return err
		}
	}
}

// skipMap consumes and discards a map's tokens (MapStart through MapEnd).
func skipMap(ts *TokenStream, valueType field.Type) error {
	tok, ok := ts.Next()
	if !ok {
		return fmt.Errorf("expected TokenMapStart, got EOF")
	}
	if tok.Kind != TokenMapStart {
		return fmt.Errorf("expected TokenMapStart, got %v", tok.Kind)
	}

	for {
		tok, ok = ts.Next()
		if !ok {
			return fmt.Errorf("unexpected EOF while skipping map")
		}
		if tok.Kind == TokenMapEnd {
			return nil
		}
		if tok.Kind != TokenMapEntry {
			return fmt.Errorf("expected TokenMapEntry, got %v", tok.Kind)
		}
		// For struct values, we need to skip the nested struct
		if valueType == field.FTStruct {
			if err := skipStruct(ts); err != nil {
				return err
			}
		}
		// For nested maps, we need to skip recursively
		if valueType == field.FTMap {
			if err := skipMap(ts, tok.ValueType); err != nil {
				return err
			}
		}
		// Scalar and string values are already contained in the MapEntry token
	}
}
