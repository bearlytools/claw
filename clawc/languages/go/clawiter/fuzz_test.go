package clawiter

import (
	"iter"
	"math"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/field"
)

// FuzzTokenSetGetBool fuzzes Token bool set/get methods.
func FuzzTokenSetGetBool(f *testing.F) {
	f.Add(true)
	f.Add(false)

	f.Fuzz(func(t *testing.T, val bool) {
		tok := Token{Type: field.FTBool}
		tok.SetBool(val)
		if tok.Bool() != val {
			t.Errorf("FuzzTokenSetGetBool: round-trip failed: got %v, want %v", tok.Bool(), val)
		}
	})
}

// FuzzTokenSetGetInt8 fuzzes Token int8 set/get methods.
func FuzzTokenSetGetInt8(f *testing.F) {
	f.Add(int8(0))
	f.Add(int8(127))
	f.Add(int8(-128))
	f.Add(int8(-1))

	f.Fuzz(func(t *testing.T, val int8) {
		tok := Token{Type: field.FTInt8}
		tok.SetInt8(val)
		if tok.Int8() != val {
			t.Errorf("FuzzTokenSetGetInt8: round-trip failed: got %d, want %d", tok.Int8(), val)
		}
	})
}

// FuzzTokenSetGetInt16 fuzzes Token int16 set/get methods.
func FuzzTokenSetGetInt16(f *testing.F) {
	f.Add(int16(0))
	f.Add(int16(32767))
	f.Add(int16(-32768))
	f.Add(int16(-1))

	f.Fuzz(func(t *testing.T, val int16) {
		tok := Token{Type: field.FTInt16}
		tok.SetInt16(val)
		if tok.Int16() != val {
			t.Errorf("FuzzTokenSetGetInt16: round-trip failed: got %d, want %d", tok.Int16(), val)
		}
	})
}

// FuzzTokenSetGetInt32 fuzzes Token int32 set/get methods.
func FuzzTokenSetGetInt32(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(2147483647))
	f.Add(int32(-2147483648))
	f.Add(int32(-1))

	f.Fuzz(func(t *testing.T, val int32) {
		tok := Token{Type: field.FTInt32}
		tok.SetInt32(val)
		if tok.Int32() != val {
			t.Errorf("FuzzTokenSetGetInt32: round-trip failed: got %d, want %d", tok.Int32(), val)
		}
	})
}

// FuzzTokenSetGetInt64 fuzzes Token int64 set/get methods.
func FuzzTokenSetGetInt64(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(9223372036854775807))
	f.Add(int64(-9223372036854775808))
	f.Add(int64(-1))

	f.Fuzz(func(t *testing.T, val int64) {
		tok := Token{Type: field.FTInt64}
		tok.SetInt64(val)
		if tok.Int64() != val {
			t.Errorf("FuzzTokenSetGetInt64: round-trip failed: got %d, want %d", tok.Int64(), val)
		}
	})
}

// FuzzTokenSetGetUint8 fuzzes Token uint8 set/get methods.
func FuzzTokenSetGetUint8(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(255))
	f.Add(uint8(128))

	f.Fuzz(func(t *testing.T, val uint8) {
		tok := Token{Type: field.FTUint8}
		tok.SetUint8(val)
		if tok.Uint8() != val {
			t.Errorf("FuzzTokenSetGetUint8: round-trip failed: got %d, want %d", tok.Uint8(), val)
		}
	})
}

// FuzzTokenSetGetUint16 fuzzes Token uint16 set/get methods.
func FuzzTokenSetGetUint16(f *testing.F) {
	f.Add(uint16(0))
	f.Add(uint16(65535))
	f.Add(uint16(32768))

	f.Fuzz(func(t *testing.T, val uint16) {
		tok := Token{Type: field.FTUint16}
		tok.SetUint16(val)
		if tok.Uint16() != val {
			t.Errorf("FuzzTokenSetGetUint16: round-trip failed: got %d, want %d", tok.Uint16(), val)
		}
	})
}

// FuzzTokenSetGetUint32 fuzzes Token uint32 set/get methods.
func FuzzTokenSetGetUint32(f *testing.F) {
	f.Add(uint32(0))
	f.Add(uint32(4294967295))
	f.Add(uint32(2147483648))

	f.Fuzz(func(t *testing.T, val uint32) {
		tok := Token{Type: field.FTUint32}
		tok.SetUint32(val)
		if tok.Uint32() != val {
			t.Errorf("FuzzTokenSetGetUint32: round-trip failed: got %d, want %d", tok.Uint32(), val)
		}
	})
}

// FuzzTokenSetGetUint64 fuzzes Token uint64 set/get methods.
func FuzzTokenSetGetUint64(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(18446744073709551615))
	f.Add(uint64(9223372036854775808))

	f.Fuzz(func(t *testing.T, val uint64) {
		tok := Token{Type: field.FTUint64}
		tok.SetUint64(val)
		if tok.Uint64() != val {
			t.Errorf("FuzzTokenSetGetUint64: round-trip failed: got %d, want %d", tok.Uint64(), val)
		}
	})
}

// FuzzTokenSetGetFloat32 fuzzes Token float32 set/get methods.
func FuzzTokenSetGetFloat32(f *testing.F) {
	f.Add(float32(0))
	f.Add(float32(3.14159))
	f.Add(float32(-3.14159))
	f.Add(float32(1e38))
	f.Add(float32(-1e38))

	f.Fuzz(func(t *testing.T, val float32) {
		tok := Token{Type: field.FTFloat32}
		tok.SetFloat32(val)
		got := tok.Float32()

		// Handle NaN specially
		if math.IsNaN(float64(val)) {
			if !math.IsNaN(float64(got)) {
				t.Errorf("FuzzTokenSetGetFloat32: expected NaN, got %v", got)
			}
			return
		}

		if got != val {
			t.Errorf("FuzzTokenSetGetFloat32: round-trip failed: got %v, want %v", got, val)
		}
	})
}

// FuzzTokenSetGetFloat64 fuzzes Token float64 set/get methods.
func FuzzTokenSetGetFloat64(f *testing.F) {
	f.Add(float64(0))
	f.Add(float64(3.141592653589793))
	f.Add(float64(-3.141592653589793))
	f.Add(float64(1e308))
	f.Add(float64(-1e308))

	f.Fuzz(func(t *testing.T, val float64) {
		tok := Token{Type: field.FTFloat64}
		tok.SetFloat64(val)
		got := tok.Float64()

		// Handle NaN specially
		if math.IsNaN(val) {
			if !math.IsNaN(got) {
				t.Errorf("FuzzTokenSetGetFloat64: expected NaN, got %v", got)
			}
			return
		}

		if got != val {
			t.Errorf("FuzzTokenSetGetFloat64: round-trip failed: got %v, want %v", got, val)
		}
	})
}

// FuzzTokenString fuzzes Token string handling.
func FuzzTokenString(f *testing.F) {
	f.Add("")
	f.Add("hello")
	f.Add("hello world")
	f.Add("unicode: 中文")
	f.Add("\x00\x01\x02")
	f.Add("with\nnewline")

	f.Fuzz(func(t *testing.T, val string) {
		tok := Token{Type: field.FTString, Bytes: []byte(val)}
		got := tok.String()
		if got != val {
			t.Errorf("FuzzTokenString: got %q, want %q", got, val)
		}
	})
}

// FuzzTokenKeySetGetInt32 fuzzes Token map key int32 set/get methods.
func FuzzTokenKeySetGetInt32(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(2147483647))
	f.Add(int32(-2147483648))

	f.Fuzz(func(t *testing.T, val int32) {
		tok := Token{KeyType: field.FTInt32}
		tok.SetKeyInt32(val)
		if tok.KeyInt32() != val {
			t.Errorf("FuzzTokenKeySetGetInt32: round-trip failed: got %d, want %d", tok.KeyInt32(), val)
		}
	})
}

// FuzzTokenKeyString fuzzes Token map key string handling.
func FuzzTokenKeyString(f *testing.F) {
	f.Add("")
	f.Add("key")
	f.Add("key with spaces")
	f.Add("unicode: 中文")

	f.Fuzz(func(t *testing.T, val string) {
		tok := Token{KeyType: field.FTString, KeyBytes: []byte(val)}
		got := tok.KeyString()
		if got != val {
			t.Errorf("FuzzTokenKeyString: got %q, want %q", got, val)
		}
	})
}

// FuzzTokenStreamPeek fuzzes TokenStream peek/next operations.
func FuzzTokenStreamPeek(f *testing.F) {
	f.Add(uint8(5))
	f.Add(uint8(0))
	f.Add(uint8(1))
	f.Add(uint8(10))

	f.Fuzz(func(t *testing.T, count uint8) {
		// Limit token count
		if count > 100 {
			count = 100
		}

		// Create a sequence of tokens
		tokens := make([]Token, count)
		for i := range tokens {
			tokens[i] = Token{Kind: TokenField, Name: "field", Type: field.FTInt32}
			tokens[i].SetInt32(int32(i))
		}

		// Create iterator
		seq := func(yield func(Token) bool) {
			for _, tok := range tokens {
				if !yield(tok) {
					return
				}
			}
		}

		ts := NewTokenStream(iter.Seq[Token](seq))
		defer ts.Close()

		// Test peek doesn't consume
		for i := 0; i < int(count); i++ {
			peeked, ok := ts.Peek()
			if !ok {
				t.Fatalf("FuzzTokenStreamPeek: Peek returned false at position %d", i)
			}
			if peeked.Int32() != int32(i) {
				t.Errorf("FuzzTokenStreamPeek: Peek got %d, want %d", peeked.Int32(), i)
			}

			// Now consume
			next, ok := ts.Next()
			if !ok {
				t.Fatalf("FuzzTokenStreamPeek: Next returned false at position %d", i)
			}
			if next.Int32() != int32(i) {
				t.Errorf("FuzzTokenStreamPeek: Next got %d, want %d", next.Int32(), i)
			}
		}

		// Verify stream is exhausted
		_, ok := ts.Next()
		if ok {
			t.Error("FuzzTokenStreamPeek: expected stream to be exhausted")
		}
	})
}

// FuzzSkipValueScalar fuzzes SkipValue for scalar types.
func FuzzSkipValueScalar(f *testing.F) {
	f.Add(int32(42))
	f.Add(int32(0))
	f.Add(int32(-1))

	f.Fuzz(func(t *testing.T, val int32) {
		// Create a field token for scalar
		fieldTok := Token{Kind: TokenField, Type: field.FTInt32}
		fieldTok.SetInt32(val)

		// Create empty stream (scalar values are already consumed)
		seq := func(yield func(Token) bool) {}
		ts := NewTokenStream(iter.Seq[Token](seq))
		defer ts.Close()

		// Should not panic
		err := SkipValue(ts, fieldTok)
		if err != nil {
			t.Errorf("FuzzSkipValueScalar: unexpected error: %v", err)
		}
	})
}

// FuzzSkipValueNilStruct fuzzes SkipValue for nil structs.
func FuzzSkipValueNilStruct(f *testing.F) {
	f.Add("TestStruct")
	f.Add("")
	f.Add("A")

	f.Fuzz(func(t *testing.T, name string) {
		// Create a nil struct field token
		fieldTok := Token{Kind: TokenField, Type: field.FTStruct, IsNil: true, StructName: name}

		// Create empty stream (nil struct has no tokens)
		seq := func(yield func(Token) bool) {}
		ts := NewTokenStream(iter.Seq[Token](seq))
		defer ts.Close()

		// Should not panic
		err := SkipValue(ts, fieldTok)
		if err != nil {
			t.Errorf("FuzzSkipValueNilStruct: unexpected error: %v", err)
		}
	})
}

// FuzzSkipValueStruct fuzzes SkipValue for non-nil structs.
func FuzzSkipValueStruct(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(3))
	f.Add(uint8(10))

	f.Fuzz(func(t *testing.T, fieldCount uint8) {
		if fieldCount > 20 {
			fieldCount = 20
		}

		// Create a struct field token
		fieldTok := Token{Kind: TokenField, Type: field.FTStruct, StructName: "TestStruct"}

		// Create token sequence with StructStart, fields, StructEnd
		var tokens []Token
		tokens = append(tokens, Token{Kind: TokenStructStart, Name: "TestStruct"})
		for i := 0; i < int(fieldCount); i++ {
			tok := Token{Kind: TokenField, Type: field.FTInt32}
			tok.SetInt32(int32(i))
			tokens = append(tokens, tok)
		}
		tokens = append(tokens, Token{Kind: TokenStructEnd, Name: "TestStruct"})

		seq := func(yield func(Token) bool) {
			for _, tok := range tokens {
				if !yield(tok) {
					return
				}
			}
		}

		ts := NewTokenStream(iter.Seq[Token](seq))
		defer ts.Close()

		// Should not panic
		err := SkipValue(ts, fieldTok)
		if err != nil {
			t.Errorf("FuzzSkipValueStruct: unexpected error: %v", err)
		}
	})
}

// FuzzSkipValueList fuzzes SkipValue for lists.
func FuzzSkipValueList(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(5))
	f.Add(uint8(10))

	f.Fuzz(func(t *testing.T, count uint8) {
		if count > 20 {
			count = 20
		}

		// Create a list field token
		fieldTok := Token{Kind: TokenField, Type: field.FTListInt32}

		// Create token sequence with ListStart, values, ListEnd
		var tokens []Token
		tokens = append(tokens, Token{Kind: TokenListStart, Type: field.FTListInt32, Len: int(count)})
		for i := 0; i < int(count); i++ {
			tok := Token{Kind: TokenField, Type: field.FTInt32}
			tok.SetInt32(int32(i))
			tokens = append(tokens, tok)
		}
		tokens = append(tokens, Token{Kind: TokenListEnd})

		seq := func(yield func(Token) bool) {
			for _, tok := range tokens {
				if !yield(tok) {
					return
				}
			}
		}

		ts := NewTokenStream(iter.Seq[Token](seq))
		defer ts.Close()

		// Should not panic
		err := SkipValue(ts, fieldTok)
		if err != nil {
			t.Errorf("FuzzSkipValueList: unexpected error: %v", err)
		}
	})
}

// FuzzSkipValueNilList fuzzes SkipValue for nil lists.
func FuzzSkipValueNilList(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(1))

	f.Fuzz(func(t *testing.T, _ uint8) {
		// Create a nil list field token
		fieldTok := Token{Kind: TokenField, Type: field.FTListInt32, IsNil: true}

		// Create empty stream (nil list has no tokens)
		seq := func(yield func(Token) bool) {}
		ts := NewTokenStream(iter.Seq[Token](seq))
		defer ts.Close()

		// Should not panic
		err := SkipValue(ts, fieldTok)
		if err != nil {
			t.Errorf("FuzzSkipValueNilList: unexpected error: %v", err)
		}
	})
}

// FuzzIngestOptions fuzzes IngestOption configurations.
func FuzzIngestOptions(f *testing.F) {
	f.Add(true)
	f.Add(false)

	f.Fuzz(func(t *testing.T, ignore bool) {
		opt := WithIgnoreUnknownFields(ignore)
		opts := IngestOptions{}

		// Should not panic
		newOpts, err := opt(opts)
		if err != nil {
			t.Errorf("FuzzIngestOptions: unexpected error: %v", err)
		}
		if newOpts.IgnoreUnknownFields != ignore {
			t.Errorf("FuzzIngestOptions: got %v, want %v", newOpts.IgnoreUnknownFields, ignore)
		}
	})
}

// FuzzTokenBytes fuzzes Token Bytes field handling.
func FuzzTokenBytes(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3, 4, 5})
	f.Add([]byte{0xFF, 0xFE, 0xFD})

	f.Fuzz(func(t *testing.T, data []byte) {
		tok := Token{Type: field.FTBytes, Bytes: data}

		// Access all token fields
		_ = tok.Kind
		_ = tok.Name
		_ = tok.Type
		_ = tok.IsNil
		_ = tok.IsEnum
		_ = tok.EnumName

		// Verify Bytes is preserved
		if len(tok.Bytes) != len(data) {
			t.Errorf("FuzzTokenBytes: length mismatch: got %d, want %d", len(tok.Bytes), len(data))
		}
	})
}
