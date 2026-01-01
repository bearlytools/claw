// Package clawtext provides functionality to marshal Claw structures to a human-readable
// text format and unmarshal text back into Claw structures.
package clawtext

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"iter"
	"strconv"
	"strings"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/johnsiilver/halfpike"
)

// textToTokens parses clawtext and yields Claw tokens.
// The input represents the contents of a struct (no outer braces needed).
func textToTokens(input string) iter.Seq[clawiter.Token] {
	return func(yield func(clawiter.Token) bool) {
		p := &textParser{
			yield: yield,
		}

		_ = halfpike.Parse(context.Background(), input, p)
	}
}

// textParser holds the state for parsing clawtext.
type textParser struct {
	yield   func(clawiter.Token) bool
	stopped bool
	err     error
}

// Validate implements halfpike.Validator.
func (p *textParser) Validate() error {
	return p.err
}

// Start is the entry point for halfpike parsing.
func (p *textParser) Start(_ context.Context, hp *halfpike.Parser) halfpike.ParseFn {
	// Emit the root struct start
	if !p.yield(clawiter.Token{Kind: clawiter.TokenStructStart, Name: "root"}) {
		p.stopped = true
		return nil
	}
	return p.parseFields
}

// parseFields parses fields at the current nesting level.
func (p *textParser) parseFields(_ context.Context, hp *halfpike.Parser) halfpike.ParseFn {
	for {
		if p.stopped {
			return nil
		}

		p.skipCommentsAndWhitespace(hp)

		line := hp.Next()
		if hp.EOF(line) {
			// End of input - emit struct end for root
			if !p.yield(clawiter.Token{Kind: clawiter.TokenStructEnd}) {
				p.stopped = true
			}
			return nil
		}

		// Check for closing brace (end of nested struct)
		if line.Items[0].Val == "}" {
			if !p.yield(clawiter.Token{Kind: clawiter.TokenStructEnd}) {
				p.stopped = true
			}
			return nil
		}

		hp.Backup()
		if err := p.parseField(hp); err != nil {
			p.err = err
			return nil
		}
	}
}

// skipCommentsAndWhitespace skips comment lines and empty lines.
func (p *textParser) skipCommentsAndWhitespace(hp *halfpike.Parser) {
	for {
		line := hp.Next()
		if hp.EOF(line) {
			hp.Backup()
			return
		}

		// Skip empty lines
		if len(line.Items) == 0 || (len(line.Items) == 1 && line.Items[0].Val == "\n") {
			continue
		}

		first := line.Items[0].Val
		// Skip single-line comments
		if strings.HasPrefix(first, "//") {
			continue
		}
		// Skip multi-line comment start (simplified - assumes comment on single line)
		if strings.HasPrefix(first, "/*") {
			// Find the end of the comment
			for {
				if strings.Contains(line.Raw, "*/") {
					break
				}
				line = hp.Next()
				if hp.EOF(line) {
					hp.Backup()
					return
				}
			}
			continue
		}

		// Not a comment or empty line, back up
		hp.Backup()
		return
	}
}

// parseField parses a single field: Name: value,
func (p *textParser) parseField(hp *halfpike.Parser) error {
	line := hp.Next()

	if len(line.Items) < 2 {
		return fmt.Errorf("[Line %d]: invalid field format, expected 'Name: value'", line.LineNum)
	}

	// The field name may include the colon (e.g., "Name:" as one token)
	fieldName := line.Items[0].Val
	valueStart := 1

	// Check if the colon is part of the field name token
	if strings.HasSuffix(fieldName, ":") {
		fieldName = strings.TrimSuffix(fieldName, ":")
	} else if len(line.Items) >= 2 && line.Items[1].Val == ":" {
		// Colon is a separate token
		valueStart = 2
	} else {
		return fmt.Errorf("[Line %d]: expected ':' after field name %q", line.LineNum, fieldName)
	}

	if valueStart >= len(line.Items) || line.Items[valueStart].Val == "\n" {
		return fmt.Errorf("[Line %d]: expected value after ':'", line.LineNum)
	}

	// Get the value token(s)
	valueToken := line.Items[valueStart].Val

	// Handle different value types
	switch {
	case valueToken == "null":
		// Null value
		tok := clawiter.Token{
			Kind:  clawiter.TokenField,
			Name:  fieldName,
			IsNil: true,
		}
		if !p.yield(tok) {
			p.stopped = true
		}

	case valueToken == "true":
		tok := clawiter.Token{
			Kind: clawiter.TokenField,
			Name: fieldName,
			Type: field.FTBool,
		}
		tok.SetBool(true)
		if !p.yield(tok) {
			p.stopped = true
		}

	case valueToken == "false":
		tok := clawiter.Token{
			Kind: clawiter.TokenField,
			Name: fieldName,
			Type: field.FTBool,
		}
		tok.SetBool(false)
		if !p.yield(tok) {
			p.stopped = true
		}

	case valueToken == "{":
		// Nested struct
		tok := clawiter.Token{
			Kind: clawiter.TokenField,
			Name: fieldName,
			Type: field.FTStruct,
		}
		if !p.yield(tok) {
			p.stopped = true
			return nil
		}
		if !p.yield(clawiter.Token{Kind: clawiter.TokenStructStart, Name: fieldName}) {
			p.stopped = true
			return nil
		}
		// Parse nested fields recursively
		return p.parseNestedStruct(hp)

	case valueToken == "[":
		// Array
		return p.parseArray(hp, fieldName, line, valueStart)

	case valueToken == "@map":
		// Map - expect { on same line or next token
		return p.parseMap(hp, fieldName, line, valueStart)

	case strings.HasPrefix(valueToken, `"`):
		// Quoted string
		s, err := p.parseQuotedString(valueToken, line, valueStart)
		if err != nil {
			return fmt.Errorf("[Line %d]: %w", line.LineNum, err)
		}
		tok := clawiter.Token{
			Kind:  clawiter.TokenField,
			Name:  fieldName,
			Type:  field.FTString,
			Bytes: []byte(s),
		}
		if !p.yield(tok) {
			p.stopped = true
		}

	case strings.HasPrefix(valueToken, "`"):
		// Raw string (backtick)
		s, err := p.parseRawString(valueToken, hp, line)
		if err != nil {
			return fmt.Errorf("[Line %d]: %w", line.LineNum, err)
		}
		tok := clawiter.Token{
			Kind:  clawiter.TokenField,
			Name:  fieldName,
			Type:  field.FTString,
			Bytes: []byte(s),
		}
		if !p.yield(tok) {
			p.stopped = true
		}

	case strings.HasPrefix(valueToken, "0x") || strings.HasPrefix(valueToken, "0X"):
		// Hex bytes
		hexStr := strings.TrimPrefix(strings.TrimPrefix(valueToken, "0x"), "0X")
		// Remove trailing comma if present
		hexStr = strings.TrimSuffix(hexStr, ",")
		data, err := hex.DecodeString(hexStr)
		if err != nil {
			return fmt.Errorf("[Line %d]: invalid hex value: %w", line.LineNum, err)
		}
		tok := clawiter.Token{
			Kind:  clawiter.TokenField,
			Name:  fieldName,
			Type:  field.FTBytes,
			Bytes: data,
		}
		if !p.yield(tok) {
			p.stopped = true
		}

	case isNumber(valueToken):
		// Number (int or float)
		numStr := strings.TrimSuffix(valueToken, ",")
		if err := p.emitNumber(fieldName, numStr, line.LineNum); err != nil {
			return err
		}

	default:
		// Assume it's an enum name (unquoted identifier)
		// The Ingester expects the enum name in Bytes for lookup
		enumName := strings.TrimSuffix(valueToken, ",")
		tok := clawiter.Token{
			Kind:     clawiter.TokenField,
			Name:     fieldName,
			IsEnum:   true,
			EnumName: enumName,
			Bytes:    []byte(enumName), // Ingester looks for string in Bytes
		}
		if !p.yield(tok) {
			p.stopped = true
		}
	}

	return nil
}

// parseNestedStruct parses the contents of a nested struct after the opening {.
func (p *textParser) parseNestedStruct(hp *halfpike.Parser) error {
	for {
		if p.stopped {
			return nil
		}

		p.skipCommentsAndWhitespace(hp)

		line := hp.Next()
		if hp.EOF(line) {
			return fmt.Errorf("unexpected EOF in nested struct")
		}

		first := line.Items[0].Val
		// Check for closing brace
		if first == "}" || strings.HasPrefix(first, "},") {
			if !p.yield(clawiter.Token{Kind: clawiter.TokenStructEnd}) {
				p.stopped = true
			}
			return nil
		}

		hp.Backup()
		if err := p.parseField(hp); err != nil {
			return err
		}
	}
}

// parseArray parses an array value.
func (p *textParser) parseArray(hp *halfpike.Parser, fieldName string, line halfpike.Line, startIdx int) error {
	// Emit the field token
	tok := clawiter.Token{
		Kind: clawiter.TokenField,
		Name: fieldName,
	}
	if !p.yield(tok) {
		p.stopped = true
		return nil
	}

	// Emit list start
	if !p.yield(clawiter.Token{Kind: clawiter.TokenListStart}) {
		p.stopped = true
		return nil
	}

	// Check if array is on single line: [1, 2, 3]
	// or multi-line
	remaining := line.Items[startIdx+1:]
	if len(remaining) > 0 && remaining[0].Val == "]" {
		// Empty array
		if !p.yield(clawiter.Token{Kind: clawiter.TokenListEnd}) {
			p.stopped = true
		}
		return nil
	}

	// Parse array elements
	return p.parseArrayElements(hp, line, startIdx+1)
}

// parseArrayElements parses the elements inside an array.
func (p *textParser) parseArrayElements(hp *halfpike.Parser, startLine halfpike.Line, startIdx int) error {
	// First, try to parse inline elements from the start line
	items := startLine.Items[startIdx:]
	idx := 0

	for idx < len(items) {
		if p.stopped {
			return nil
		}

		val := items[idx].Val
		if val == "]" || strings.HasPrefix(val, "],") {
			if !p.yield(clawiter.Token{Kind: clawiter.TokenListEnd}) {
				p.stopped = true
			}
			return nil
		}

		if val == "," || val == "\n" {
			idx++
			continue
		}

		if val == "{" {
			// Struct in array
			if !p.yield(clawiter.Token{Kind: clawiter.TokenStructStart, Name: ""}) {
				p.stopped = true
				return nil
			}
			if err := p.parseNestedStruct(hp); err != nil {
				return err
			}
			idx++
			continue
		}

		if err := p.emitArrayElement(val, startLine.LineNum); err != nil {
			return err
		}
		idx++
	}

	// Continue parsing from subsequent lines
	for {
		if p.stopped {
			return nil
		}

		p.skipCommentsAndWhitespace(hp)

		line := hp.Next()
		if hp.EOF(line) {
			return fmt.Errorf("unexpected EOF in array")
		}

		for _, item := range line.Items {
			if p.stopped {
				return nil
			}

			val := item.Val
			if val == "]" || strings.HasPrefix(val, "],") {
				if !p.yield(clawiter.Token{Kind: clawiter.TokenListEnd}) {
					p.stopped = true
				}
				return nil
			}

			if val == "," || val == "\n" {
				continue
			}

			if val == "{" {
				// Struct in array
				if !p.yield(clawiter.Token{Kind: clawiter.TokenStructStart, Name: ""}) {
					p.stopped = true
					return nil
				}
				if err := p.parseNestedStruct(hp); err != nil {
					return err
				}
				continue
			}

			if err := p.emitArrayElement(val, line.LineNum); err != nil {
				return err
			}
		}
	}
}

// emitArrayElement emits a token for a single array element.
func (p *textParser) emitArrayElement(val string, lineNum int) error {
	val = strings.TrimSuffix(val, ",")

	switch {
	case val == "true":
		tok := clawiter.Token{Kind: clawiter.TokenField, Type: field.FTBool}
		tok.SetBool(true)
		if !p.yield(tok) {
			p.stopped = true
		}
	case val == "false":
		tok := clawiter.Token{Kind: clawiter.TokenField, Type: field.FTBool}
		tok.SetBool(false)
		if !p.yield(tok) {
			p.stopped = true
		}
	case strings.HasPrefix(val, `"`):
		s, err := unquoteString(val)
		if err != nil {
			return fmt.Errorf("[Line %d]: invalid string: %w", lineNum, err)
		}
		tok := clawiter.Token{Kind: clawiter.TokenField, Type: field.FTString, Bytes: []byte(s)}
		if !p.yield(tok) {
			p.stopped = true
		}
	case isNumber(val):
		return p.emitNumber("", val, lineNum)
	default:
		// Enum value - Ingester expects string in Bytes
		tok := clawiter.Token{Kind: clawiter.TokenField, IsEnum: true, EnumName: val, Bytes: []byte(val)}
		if !p.yield(tok) {
			p.stopped = true
		}
	}
	return nil
}

// parseMap parses a map value: @map { "key": value, ... }
func (p *textParser) parseMap(hp *halfpike.Parser, fieldName string, line halfpike.Line, startIdx int) error {
	// Expect { after @map
	braceIdx := startIdx + 1
	if braceIdx >= len(line.Items) || line.Items[braceIdx].Val != "{" {
		return fmt.Errorf("[Line %d]: expected '{' after @map", line.LineNum)
	}

	// Emit the field token for the map
	tok := clawiter.Token{
		Kind: clawiter.TokenField,
		Name: fieldName,
		Type: field.FTMap,
	}
	if !p.yield(tok) {
		p.stopped = true
		return nil
	}

	// Emit map start
	if !p.yield(clawiter.Token{Kind: clawiter.TokenMapStart}) {
		p.stopped = true
		return nil
	}

	// Parse map entries
	return p.parseMapEntries(hp)
}

// parseMapEntries parses the entries inside a map.
func (p *textParser) parseMapEntries(hp *halfpike.Parser) error {
	for {
		if p.stopped {
			return nil
		}

		p.skipCommentsAndWhitespace(hp)

		line := hp.Next()
		if hp.EOF(line) {
			return fmt.Errorf("unexpected EOF in map")
		}

		first := line.Items[0].Val
		// Check for closing brace
		if first == "}" || strings.HasPrefix(first, "},") {
			if !p.yield(clawiter.Token{Kind: clawiter.TokenMapEnd}) {
				p.stopped = true
			}
			return nil
		}

		// Parse map entry: "key": value
		if !strings.HasPrefix(first, `"`) {
			return fmt.Errorf("[Line %d]: map key must be a quoted string, got %q", line.LineNum, first)
		}

		key, err := unquoteString(first)
		if err != nil {
			return fmt.Errorf("[Line %d]: invalid map key: %w", line.LineNum, err)
		}

		// Expect colon
		if len(line.Items) < 2 || line.Items[1].Val != ":" {
			return fmt.Errorf("[Line %d]: expected ':' after map key", line.LineNum)
		}

		if len(line.Items) < 3 {
			return fmt.Errorf("[Line %d]: expected value after ':'", line.LineNum)
		}

		// Get value
		valueToken := line.Items[2].Val
		valueToken = strings.TrimSuffix(valueToken, ",")

		// Emit map entry
		entry := clawiter.Token{
			Kind:     clawiter.TokenMapEntry,
			KeyBytes: []byte(key),
		}

		// Set value in token based on type
		switch {
		case strings.HasPrefix(valueToken, `"`):
			s, err := unquoteString(valueToken)
			if err != nil {
				return fmt.Errorf("[Line %d]: invalid string value: %w", line.LineNum, err)
			}
			entry.Type = field.FTString
			entry.Bytes = []byte(s)
		case valueToken == "true":
			entry.Type = field.FTBool
			entry.SetBool(true)
		case valueToken == "false":
			entry.Type = field.FTBool
			entry.SetBool(false)
		case isNumber(valueToken):
			if strings.Contains(valueToken, ".") || strings.Contains(valueToken, "e") || strings.Contains(valueToken, "E") {
				f, err := strconv.ParseFloat(valueToken, 64)
				if err != nil {
					return fmt.Errorf("[Line %d]: invalid float: %w", line.LineNum, err)
				}
				entry.Type = field.FTFloat64
				entry.SetFloat64(f)
			} else {
				n, err := strconv.ParseInt(valueToken, 10, 64)
				if err != nil {
					return fmt.Errorf("[Line %d]: invalid integer: %w", line.LineNum, err)
				}
				entry.SetInt64(n)
			}
		case valueToken == "{":
			// Struct value in map
			entry.Type = field.FTStruct
			if !p.yield(entry) {
				p.stopped = true
				return nil
			}
			if !p.yield(clawiter.Token{Kind: clawiter.TokenStructStart, Name: ""}) {
				p.stopped = true
				return nil
			}
			if err := p.parseNestedStruct(hp); err != nil {
				return err
			}
			continue
		default:
			// Assume enum - Ingester expects string in Bytes
			entry.IsEnum = true
			entry.EnumName = valueToken
			entry.Bytes = []byte(valueToken)
		}

		if !p.yield(entry) {
			p.stopped = true
			return nil
		}
	}
}

// parseQuotedString parses a double-quoted string, handling escapes.
func (p *textParser) parseQuotedString(token string, line halfpike.Line, startIdx int) (string, error) {
	// The token might be the complete string or just the start
	// Check if it ends with a quote (not counting trailing comma)
	s := token
	s = strings.TrimSuffix(s, ",")

	if strings.HasSuffix(s, `"`) && len(s) > 1 {
		// Complete string in one token
		return unquoteString(s)
	}

	// String might span multiple tokens on the same line (with spaces)
	// Reconstruct from raw line
	raw := line.Raw
	colonIdx := strings.Index(raw, ":")
	if colonIdx == -1 {
		return "", fmt.Errorf("malformed field")
	}
	valueStr := strings.TrimSpace(raw[colonIdx+1:])
	valueStr = strings.TrimSuffix(valueStr, ",")
	return unquoteString(valueStr)
}

// parseRawString parses a backtick-delimited raw string.
func (p *textParser) parseRawString(token string, hp *halfpike.Parser, line halfpike.Line) (string, error) {
	// Check if the string is complete on this line
	if strings.Count(token, "`") >= 2 {
		// Complete on one token
		return strings.Trim(token, "`"), nil
	}

	// Multi-line raw string - need to read until closing backtick
	var sb strings.Builder
	sb.WriteString(strings.TrimPrefix(token, "`"))

	for {
		nextLine := hp.Next()
		if hp.EOF(nextLine) {
			return "", fmt.Errorf("unexpected EOF in raw string")
		}

		raw := nextLine.Raw
		if idx := strings.Index(raw, "`"); idx != -1 {
			// Found closing backtick
			sb.WriteString("\n")
			sb.WriteString(raw[:idx])
			return sb.String(), nil
		}

		sb.WriteString("\n")
		sb.WriteString(strings.TrimSuffix(raw, "\n"))
	}
}

// emitNumber parses and emits a numeric token.
func (p *textParser) emitNumber(name, numStr string, lineNum int) error {
	numStr = strings.TrimSuffix(numStr, ",")

	tok := clawiter.Token{
		Kind: clawiter.TokenField,
		Name: name,
	}

	if strings.Contains(numStr, ".") || strings.Contains(numStr, "e") || strings.Contains(numStr, "E") {
		// Float
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return fmt.Errorf("[Line %d]: invalid float %q: %w", lineNum, numStr, err)
		}
		tok.Type = field.FTFloat64
		tok.SetFloat64(f)
	} else {
		// Integer
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return fmt.Errorf("[Line %d]: invalid integer %q: %w", lineNum, numStr, err)
		}
		tok.SetInt64(n)
	}

	if !p.yield(tok) {
		p.stopped = true
	}
	return nil
}

// isNumber checks if the string looks like a number.
func isNumber(s string) bool {
	s = strings.TrimSuffix(s, ",")
	if len(s) == 0 {
		return false
	}
	// Check for leading sign
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
		if len(s) == 1 {
			return false
		}
	}
	// Must start with digit
	if s[start] < '0' || s[start] > '9' {
		return false
	}
	return true
}

// unquoteString removes quotes and handles escape sequences.
func unquoteString(s string) (string, error) {
	// Remove surrounding quotes
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("string must be quoted")
	}
	s = s[1 : len(s)-1]

	// Handle escape sequences
	var sb strings.Builder
	sb.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '"':
				sb.WriteByte('"')
				i++
			case '\\':
				sb.WriteByte('\\')
				i++
			case 'n':
				sb.WriteByte('\n')
				i++
			case 'r':
				sb.WriteByte('\r')
				i++
			case 't':
				sb.WriteByte('\t')
				i++
			case 'u':
				// Unicode escape: \uXXXX
				if i+5 < len(s) {
					hex := s[i+2 : i+6]
					code, err := strconv.ParseUint(hex, 16, 32)
					if err != nil {
						return "", fmt.Errorf("invalid unicode escape: \\u%s", hex)
					}
					sb.WriteRune(rune(code))
					i += 5
				} else {
					return "", fmt.Errorf("incomplete unicode escape")
				}
			default:
				sb.WriteByte(s[i])
			}
		} else {
			sb.WriteByte(s[i])
		}
	}

	return sb.String(), nil
}

// decodeBytes decodes a base64 string to bytes.
func decodeBytes(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
