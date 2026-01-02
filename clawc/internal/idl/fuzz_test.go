package idl

import (
	"testing"
)

// FuzzValidPackage fuzzes the package name validation function.
func FuzzValidPackage(f *testing.F) {
	// Valid package names
	f.Add("mypackage")
	f.Add("pkg")
	f.Add("my_package")
	f.Add("pkg123")
	f.Add("a")
	f.Add("abc_def_123")

	// Invalid package names
	f.Add("")
	f.Add("MyPackage")     // starts with uppercase
	f.Add("123pkg")        // starts with number
	f.Add("_pkg")          // starts with underscore
	f.Add("pkg-name")      // contains hyphen
	f.Add("pkg.name")      // contains dot
	f.Add("pkg name")      // contains space
	f.Add("pkg\tname")     // contains tab
	f.Add("pkg\nname")     // contains newline
	f.Add("pkg@name")      // contains @
	f.Add("中文")           // unicode

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) == 0 {
			return
		}
		// Should not panic
		_ = ValidPackage(input)
	})
}

// FuzzValidateIdent fuzzes the identifier validation function.
func FuzzValidateIdent(f *testing.F) {
	// Valid identifiers
	f.Add("MyIdent")
	f.Add("A")
	f.Add("ABC")
	f.Add("MyStruct123")
	f.Add("Type")
	f.Add("UPPER")

	// Invalid identifiers
	f.Add("")
	f.Add("myident")       // starts with lowercase
	f.Add("123Type")       // starts with number
	f.Add("_Type")         // starts with underscore
	f.Add("My-Type")       // contains hyphen
	f.Add("My.Type")       // contains dot
	f.Add("My Type")       // contains space
	f.Add("My_Type")       // contains underscore (still valid)
	f.Add("@Annotation")   // starts with @
	f.Add("中文")           // unicode

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) == 0 {
			return
		}
		// Should not panic
		_ = validateIdent(input)
	})
}

// FuzzCaseSensitiveCheck fuzzes the case-sensitive keyword checking function.
func FuzzCaseSensitiveCheck(f *testing.F) {
	// Correct keywords
	f.Add("package", "package")
	f.Add("version", "version")
	f.Add("struct", "struct")
	f.Add("enum", "enum")

	// Wrong case
	f.Add("package", "Package")
	f.Add("package", "PACKAGE")
	f.Add("version", "Version")
	f.Add("struct", "Struct")

	// Completely different
	f.Add("package", "something")
	f.Add("version", "")
	f.Add("struct", "enum")

	f.Fuzz(func(t *testing.T, want, got string) {
		// Should not panic
		_ = caseSensitiveCheck(want, got)
	})
}

// FuzzParseScalarType fuzzes the scalar type parsing function.
func FuzzParseScalarType(f *testing.F) {
	// Valid scalar types
	f.Add("bool")
	f.Add("int8")
	f.Add("int16")
	f.Add("int32")
	f.Add("int64")
	f.Add("uint8")
	f.Add("uint16")
	f.Add("uint32")
	f.Add("uint64")
	f.Add("float32")
	f.Add("float64")
	f.Add("string")
	f.Add("bytes")

	// Invalid types
	f.Add("")
	f.Add("int")
	f.Add("uint")
	f.Add("float")
	f.Add("double")
	f.Add("char")
	f.Add("Int32")        // wrong case
	f.Add("STRING")       // wrong case
	f.Add("[]int32")      // list type
	f.Add("map[int]int")  // map type
	f.Add("MyStruct")     // struct type
	f.Add("123")          // number
	f.Add(" int32")       // leading space

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic
		_, _ = parseScalarType(input)
	})
}

// FuzzValidImportPath fuzzes the import path validation function.
func FuzzValidImportPath(f *testing.F) {
	// Valid import paths
	f.Add(`"github.com/pkg/errors"`)
	f.Add(`"example.com/my/package"`)
	f.Add(`"internal/pkg"`)
	f.Add(`"a/b"`)

	// Invalid import paths
	f.Add(`github.com/pkg`)       // no quotes
	f.Add(`"github.com/pkg`)      // missing end quote
	f.Add(`github.com/pkg"`)      // missing start quote
	f.Add(`""`)                   // empty path
	f.Add(`"pkg"`)                // no slash (single component)
	f.Add(`"pkg/"`)               // trailing slash
	f.Add(`"/pkg/a"`)             // leading slash
	f.Add(`"pkg//a"`)             // double slash
	f.Add(`"."`)                  // just dot
	f.Add(`".."`)                 // just dots

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic
		_, _ = validImportPath(input)
	})
}

// FuzzRemoveSpace fuzzes the space removal helper function.
func FuzzRemoveSpace(f *testing.F) {
	f.Add("hello")
	f.Add("  hello")
	f.Add("\t\thello")
	f.Add("\n\nhello")
	f.Add("   \t\n  hello")
	f.Add("")
	f.Add("   ")
	f.Add("\t\t\t")
	f.Add("no leading space")
	f.Add("  mixed \t content")

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic
		_ = removeSpace([]rune(input))
	})
}
