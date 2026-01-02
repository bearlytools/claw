package clawtext

import (
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
)

// FuzzTextToTokens fuzzes the text parser which processes human-readable clawtext format.
// This tests parsing of field names, values, nested structs, arrays, maps, and escape sequences.
func FuzzTextToTokens(f *testing.F) {
	// Seed with valid inputs
	f.Add(`Name: "hello",`)
	f.Add(`Count: 42,`)
	f.Add(`Active: true,`)
	f.Add(`Disabled: false,`)
	f.Add(`Data: null,`)
	f.Add(`Values: [1, 2, 3],`)
	f.Add(`Names: ["a", "b", "c"],`)
	f.Add(`Nested: {
    Field: 123,
},`)
	f.Add(`Map: @map {
    "key": "value",
},`)
	f.Add(`Hex: 0xDEADBEEF,`)
	f.Add(`Float: 3.14159,`)
	f.Add(`Negative: -42,`)
	f.Add(`Scientific: 1.5e10,`)

	// Seed with edge cases
	f.Add(`Name: "",`)                  // empty string
	f.Add(`Name: "hello\nworld",`)      // escape sequence
	f.Add(`Name: "test\u0041",`)        // unicode escape
	f.Add(`Name: "quote\"here",`)       // escaped quote
	f.Add(`Name: "backslash\\here",`)   // escaped backslash
	f.Add("Name: `raw string`,")        // raw string
	f.Add(`Values: [],`)                // empty array
	f.Add(`Map: @map {},`)              // empty map
	f.Add(`// comment`)                 // comment only
	f.Add(`/* multi-line */`)           // multi-line comment
	f.Add(`Enum: SomeValue,`)           // enum value
	f.Add(`Deep: { Inner: { Leaf: 1 }}`) // deeply nested

	// Any type format seeds
	f.Add(`Data: @any(Inner) { ID: 123, Value: "test" },`)
	f.Add(`Data: @any(Outer) { Name: "outer", Inner: @any(Inner) { ID: 1 } },`)
	f.Add(`Data: @any(0x12345678, "SGVsbG8gV29ybGQ="),`) // unknown type with base64
	f.Add(`Items: [@any(Inner) { ID: 1 }, @any(Inner) { ID: 2 }],`) // list of any
	f.Add(`Data: null,`) // null any
	f.Add(`Data: @any(Empty) {},`) // empty any struct

	// Invalid any type formats
	f.Add(`Data: @any(`)        // incomplete
	f.Add(`Data: @any()`)       // empty type name
	f.Add(`Data: @any(Name`)    // missing closing paren
	f.Add(`Data: @any(Name) {`) // unclosed struct
	f.Add(`Data: @any(0x, "data"),`) // incomplete hex
	f.Add(`Data: @any(0xGGGG, "data"),`) // invalid hex
	f.Add(`Data: @any(0x1234, ),`) // missing base64

	// Seed with potentially problematic inputs
	f.Add(``)                           // empty
	f.Add(`{`)                          // unclosed brace
	f.Add(`}`)                          // unmatched closing brace
	f.Add(`[`)                          // unclosed bracket
	f.Add(`]`)                          // unmatched closing bracket
	f.Add(`Name:`)                      // missing value
	f.Add(`Name`)                       // missing colon
	f.Add(`: value`)                    // missing name
	f.Add(`Name: "unterminated`)        // unterminated string
	f.Add(`Name: "bad\uescape"`)        // invalid unicode escape
	f.Add(`Name: "\u"`)                 // incomplete unicode
	f.Add(`Name: "\uGGGG"`)             // invalid hex in unicode
	f.Add(`Values: [1, 2,`)             // unclosed array
	f.Add(`Map: @map { "key":`)         // incomplete map entry
	f.Add(`Name: 0x`)                   // incomplete hex
	f.Add(`Name: 0xGG`)                 // invalid hex
	f.Add(`Name: -`)                    // incomplete negative number
	f.Add(`Name: 1e`)                   // incomplete scientific notation
	f.Add("Name: `unterminated")        // unterminated raw string
	f.Add(`Deeply: { Nested: { Even: { More: { Deep: { Here: 1 }}}}}`) // very deep nesting

	f.Fuzz(func(t *testing.T, input string) {
		tokens := textToTokens(input)

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

// FuzzUnquoteString fuzzes the string unquoting function which handles escape sequences.
func FuzzUnquoteString(f *testing.F) {
	// Valid quoted strings
	f.Add(`"hello"`)
	f.Add(`""`)
	f.Add(`"hello world"`)
	f.Add(`"with\nnewline"`)
	f.Add(`"with\ttab"`)
	f.Add(`"with\rcarriage"`)
	f.Add(`"with\"quote"`)
	f.Add(`"with\\backslash"`)
	f.Add(`"\u0041"`)     // 'A'
	f.Add(`"\u4e2d"`)     // Chinese character
	f.Add(`"\u0000"`)     // null character
	f.Add(`"\uFFFF"`)     // max BMP
	f.Add(`"mixed\u0041\n\t"`)

	// Invalid/edge cases
	f.Add(`hello`)        // not quoted
	f.Add(`"`)            // single quote
	f.Add(`"unterminated`)
	f.Add(`""extra`)      // extra chars after
	f.Add(`"bad\escape"`) // invalid escape
	f.Add(`"\u"`)         // incomplete unicode
	f.Add(`"\uXXXX"`)     // invalid hex
	f.Add(`"\u00"`)       // too short unicode
	f.Add(`"\"`)          // escaped quote at end

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic
		_, _ = unquoteString(input)
	})
}

// FuzzIsNumber fuzzes the number detection function.
func FuzzIsNumber(f *testing.F) {
	// Valid numbers
	f.Add("0")
	f.Add("1")
	f.Add("123")
	f.Add("-1")
	f.Add("+1")
	f.Add("0,")   // with trailing comma
	f.Add("-42,")
	f.Add("3.14")
	f.Add("-3.14")
	f.Add("1e10")
	f.Add("1E10")
	f.Add("1.5e-10")

	// Invalid numbers
	f.Add("")
	f.Add("-")
	f.Add("+")
	f.Add("abc")
	f.Add("12abc")
	f.Add("--1")
	f.Add("++1")
	f.Add(".")
	f.Add(".5")

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic
		_ = isNumber(input)
	})
}

// FuzzParseRoundTrip tests that parsing and then consuming tokens doesn't crash
// with various combinations of field types.
func FuzzParseRoundTrip(f *testing.F) {
	// Multi-field inputs
	f.Add(`Name: "test",
Count: 42,
Active: true,`)
	f.Add(`Data: {
    Inner: 123,
    Nested: {
        Deep: "value",
    },
},`)
	f.Add(`Items: [
    { Name: "first" },
    { Name: "second" },
],`)
	f.Add(`Config: @map {
    "host": "localhost",
    "port": 8080,
},`)

	f.Fuzz(func(t *testing.T, input string) {
		tokens := textToTokens(input)

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
