package clawjson

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
)

// FuzzJSONToTokens fuzzes the JSON parser which converts JSON to Claw tokens.
// This tests handling of nested objects, arrays, strings, numbers, booleans, and nulls.
func FuzzJSONToTokens(f *testing.F) {
	// Valid JSON inputs
	f.Add(`{}`)
	f.Add(`{"name": "value"}`)
	f.Add(`{"count": 42}`)
	f.Add(`{"active": true}`)
	f.Add(`{"disabled": false}`)
	f.Add(`{"data": null}`)
	f.Add(`{"values": [1, 2, 3]}`)
	f.Add(`{"names": ["a", "b", "c"]}`)
	f.Add(`{"nested": {"inner": 123}}`)
	f.Add(`{"float": 3.14159}`)
	f.Add(`{"negative": -42}`)
	f.Add(`{"scientific": 1.5e10}`)
	f.Add(`{"mixed": [1, "two", true, null]}`)

	// Unicode and escape sequences
	f.Add(`{"unicode": "hello \u0041 world"}`)
	f.Add(`{"chinese": "\u4e2d\u6587"}`)
	f.Add(`{"emoji": "😀"}`)
	f.Add(`{"newline": "line1\nline2"}`)
	f.Add(`{"tab": "col1\tcol2"}`)
	f.Add(`{"quote": "say \"hello\""}`)
	f.Add(`{"backslash": "path\\to\\file"}`)
	f.Add(`{"slash": "a\/b"}`)
	f.Add(`{"all": "\"\\\n\r\t"}`)

	// Deeply nested
	f.Add(`{"a": {"b": {"c": {"d": {"e": 1}}}}}`)
	f.Add(`[[[[[[1]]]]]]`)

	// Any type format seeds (JSON representation)
	f.Add(`{"Data": {"@type": "example.com/pkg.Inner", "@fieldType": "Inner", "ID": 123, "Value": "test"}}`)
	f.Add(`{"Data": {"@type": "example.com/pkg.Outer", "@fieldType": "Outer", "Name": "outer"}}`)
	f.Add(`{"Items": [{"@type": "pkg.Inner", "@fieldType": "Inner", "ID": 1}, {"@type": "pkg.Inner", "@fieldType": "Inner", "ID": 2}]}`)
	f.Add(`{"Data": null}`) // null any
	f.Add(`{"Data": {"@type": "pkg.Empty", "@fieldType": "Empty"}}`) // empty any struct

	// Invalid/edge case any type formats
	f.Add(`{"Data": {"@type": ""}}`)         // empty type
	f.Add(`{"Data": {"@fieldType": ""}}`)    // empty field type
	f.Add(`{"Data": {"@type": 123}}`)        // non-string type
	f.Add(`{"Data": {"@fieldType": null}}`)  // null field type

	// Edge cases
	f.Add(`[]`)                        // empty array
	f.Add(`[{}]`)                      // array with empty object
	f.Add(`{"": "empty key"}`)         // empty key
	f.Add(`{"key": ""}`)               // empty value
	f.Add(`{"a": 0}`)                  // zero
	f.Add(`{"a": -0}`)                 // negative zero
	f.Add(`{"a": 0.0}`)                // zero float
	f.Add(`{"a": 1e100}`)              // large exponent
	f.Add(`{"a": 1e-100}`)             // small exponent
	f.Add(`{"a": 9223372036854775807}`) // int64 max

	// Potentially problematic inputs (invalid JSON)
	f.Add(``)                          // empty
	f.Add(`{`)                         // unclosed brace
	f.Add(`}`)                         // unmatched closing brace
	f.Add(`[`)                         // unclosed bracket
	f.Add(`]`)                         // unmatched closing bracket
	f.Add(`{"key"}`)                   // missing value
	f.Add(`{"key":}`)                  // empty value
	f.Add(`{key: "value"}`)            // unquoted key
	f.Add(`{"key": "unterminated`)     // unterminated string
	f.Add(`{"key": undefined}`)        // undefined (not valid JSON)
	f.Add(`{"key": NaN}`)              // NaN literal
	f.Add(`{"key": Infinity}`)         // Infinity literal
	f.Add(`{"a": 1,}`)                 // trailing comma
	f.Add(`[1, 2,]`)                   // trailing comma in array
	f.Add(`{"a": 1, "a": 2}`)          // duplicate keys
	f.Add(`{"key": "\uD800"}`)         // lone surrogate
	f.Add(`{"key": "\uDC00"}`)         // lone low surrogate
	f.Add(`{"key": "\uD800\uD800"}`)   // double high surrogate

	f.Fuzz(func(t *testing.T, input string) {
		tokens := jsonToTokens(strings.NewReader(input))

		// The parser should not panic on any input
		// We consume all tokens to ensure the full parsing path is exercised
		for tok := range tokens {
			// Access token fields to ensure they're valid
			_ = tok.Kind
			_ = tok.Name
			_ = tok.Type
			_ = tok.IsNil
			_ = tok.IsEnum
			_ = tok.EnumName
			_ = tok.Bytes
		}
	})
}

// FuzzAppendEscapedString fuzzes the string escaping function for JSON output.
func FuzzAppendEscapedString(f *testing.F) {
	// Normal strings
	f.Add("hello")
	f.Add("")
	f.Add("hello world")

	// Strings needing escaping
	f.Add("hello\nworld")
	f.Add("hello\tworld")
	f.Add("hello\rworld")
	f.Add(`hello"world`)
	f.Add(`hello\world`)

	// Control characters
	f.Add("\x00")
	f.Add("\x01")
	f.Add("\x1f")

	// Unicode
	f.Add("中文")
	f.Add("日本語")
	f.Add("😀🎉")
	f.Add("\u0000")
	f.Add("\u001f")

	// High bytes
	f.Add("\x80")
	f.Add("\xff")

	// Mixed
	f.Add("hello\n\"world\"\t中文")

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic
		result := appendEscapedString(nil, input)

		// Result should always start and end with quotes
		if len(result) < 2 {
			t.Errorf("FuzzAppendEscapedString: result too short: %q", result)
			return
		}
		if result[0] != '"' || result[len(result)-1] != '"' {
			t.Errorf("FuzzAppendEscapedString: result not quoted: %q", result)
		}
	})
}

// FuzzStringToBytes fuzzes the unsafe string to bytes conversion.
func FuzzStringToBytes(f *testing.F) {
	f.Add("")
	f.Add("hello")
	f.Add("hello world")
	f.Add("unicode: 中文")
	f.Add("\x00\x01\x02")

	f.Fuzz(func(t *testing.T, input string) {
		result := stringToBytes(input)

		// Result length should match input
		if len(result) != len(input) {
			t.Errorf("FuzzStringToBytes: length mismatch: got %d, want %d", len(result), len(input))
		}

		// Content should match (via comparison)
		if string(result) != input {
			t.Errorf("FuzzStringToBytes: content mismatch")
		}
	})
}

// FuzzJSONRoundTrip tests that parsing JSON and consuming all tokens works
// without crashing for various JSON structures.
func FuzzJSONRoundTrip(f *testing.F) {
	// Multi-field objects
	f.Add(`{
		"name": "test",
		"count": 42,
		"active": true
	}`)
	f.Add(`{
		"data": {
			"inner": 123,
			"nested": {
				"deep": "value"
			}
		}
	}`)
	f.Add(`{
		"items": [
			{"name": "first"},
			{"name": "second"}
		]
	}`)

	f.Fuzz(func(t *testing.T, input string) {
		tokens := jsonToTokens(strings.NewReader(input))

		var tokenList []clawiter.Token
		for tok := range tokens {
			tokenList = append(tokenList, tok)
		}

		// Verify we can safely access all token data
		for _, tok := range tokenList {
			switch tok.Kind {
			case clawiter.TokenStructStart, clawiter.TokenStructEnd:
				_ = tok.Name
			case clawiter.TokenListStart, clawiter.TokenListEnd:
				// No additional fields
			case clawiter.TokenMapStart, clawiter.TokenMapEnd:
				// No additional fields
			case clawiter.TokenField:
				_ = tok.Name
				_ = tok.Type
				_ = tok.IsNil
				_ = tok.IsEnum
				_ = tok.EnumName
				_ = len(tok.Bytes)
			case clawiter.TokenMapEntry:
				_ = len(tok.KeyBytes)
				_ = tok.Type
			}
		}
	})
}

// FuzzJSONBytes tests parsing JSON with arbitrary bytes (may contain invalid UTF-8).
func FuzzJSONBytes(f *testing.F) {
	// Valid JSON as bytes
	f.Add([]byte(`{"key": "value"}`))
	f.Add([]byte(`{"count": 42}`))
	f.Add([]byte(`[1, 2, 3]`))

	// Invalid UTF-8 sequences in JSON
	f.Add([]byte{'{', '"', 'k', '"', ':', '"', 0x80, '"', '}'})
	f.Add([]byte{'{', '"', 'k', '"', ':', '"', 0xff, '"', '}'})
	f.Add([]byte{'{', '"', 0xc0, 0x80, '"', ':', '"', 'v', '"', '}'}) // overlong encoding

	// Truncated UTF-8
	f.Add([]byte{'{', '"', 'k', '"', ':', '"', 0xc2, '"', '}'}) // incomplete 2-byte
	f.Add([]byte{'{', '"', 'k', '"', ':', '"', 0xe0, 0xa0, '"', '}'}) // incomplete 3-byte

	f.Fuzz(func(t *testing.T, input []byte) {
		tokens := jsonToTokens(bytes.NewReader(input))

		// Should not panic
		for tok := range tokens {
			_ = tok.Kind
			_ = tok.Name
			_ = tok.Type
		}
	})
}
