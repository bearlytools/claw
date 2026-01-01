# Clawtext Format

Clawtext is a human-readable text format for Claw data. It is designed to be easy to read, write, and edit by hand while avoiding common pitfalls found in formats like YAML.

## Design Principles

1. **Explicit over implicit** - No implicit type coercion or magic conversions
2. **Readable** - Clear syntax with support for comments and flexible whitespace
3. **Unambiguous** - Strings are always quoted, booleans are only `true`/`false`
4. **Familiar** - Syntax similar to JSON but with improvements for human editing

## File Extension

Clawtext files use the `.clawtext` extension.

## Basic Syntax

### Top-Level Structure

A clawtext file represents a single Claw struct. Unlike JSON, no outer braces are required - fields are written directly at the top level:

```clawtext
Name: "web-server-01",
Port: 8080,
Active: true,
```

### Field Format

Fields use the format `FieldName: value,` where:
- Field names are case-sensitive
- A colon (`:`) separates the name from the value
- A comma (`,`) separates fields
- Trailing comma on the last field is optional

### Comments

Both single-line and multi-line comments are supported:

```clawtext
// This is a single-line comment
Name: "server",  // Inline comment

/* This is a
   multi-line comment */
Port: 8080,
```

### Whitespace

Whitespace (spaces, tabs, newlines) is flexible and ignored between tokens. These are equivalent:

```clawtext
Name: "server", Port: 8080,
```

```clawtext
Name: "server",
Port: 8080,
```

## Data Types

### Strings

Strings are always enclosed in double quotes. Standard escape sequences are supported:

```clawtext
Message: "Hello, World!",
Path: "C:\\Users\\name",
Quote: "She said \"hello\"",
```

For multi-line strings, use backticks:

```clawtext
Description: `This is a
multi-line string that preserves
line breaks`,
```

### Numbers

Integers and floating-point numbers are written as literals:

```clawtext
Count: 42,
Price: 19.99,
Negative: -100,
Scientific: 1.5e10,
```

Hexadecimal integers use the `0x` prefix:

```clawtext
Flags: 0xFF,
```

### Booleans

Only `true` and `false` are valid boolean values. Unlike YAML, words like "yes", "no", "on", "off" are not treated as booleans.

```clawtext
Enabled: true,
Debug: false,
```

### Null

Use `null` to represent unset or nil values:

```clawtext
OptionalField: null,
```

### Enums

Enum values are written by name (not number):

```clawtext
Status: Running,
Type: Car,
Manufacturer: Toyota,
```

When marshaling, you can optionally use enum numbers with the `WithUseEnumNumbers(true)` option:

```clawtext
Status: 1,
Type: 1,
Manufacturer: 1,
```

### Bytes

Byte arrays can be encoded as base64 strings or hexadecimal with `0x` prefix:

```clawtext
// Base64 encoding (default)
Data: "SGVsbG8gV29ybGQ=",

// Hexadecimal encoding
Data: 0x48656C6C6F20576F726C64,
```

When marshaling, use `WithUseHexBytes(true)` to output hex format.

## Complex Types

### Nested Structs

Nested structs use curly braces:

```clawtext
Config: {
    MaxConnections: 1000,
    Timeout: 30,
    Retry: {
        Attempts: 3,
        Delay: 5,
    },
},
```

### Arrays/Lists

Arrays use square brackets with comma-separated values:

```clawtext
// Array of numbers
Ports: [8080, 8081, 8082],

// Array of strings
Tags: ["production", "web", "primary"],

// Array of booleans
Flags: [true, false, true],

// Array of enums
Types: [Car, Truck, Motorcycle],
```

### Arrays of Structs

```clawtext
Servers: [
    {
        Name: "server-1",
        Port: 8080,
    },
    {
        Name: "server-2",
        Port: 8081,
    },
],
```

### Maps

Maps use the `@map` annotation to distinguish them from structs:

```clawtext
Labels: @map {
    "env": "production",
    "region": "us-west-2",
    "team": "platform",
},
```

## Complete Example

```clawtext
// Server configuration file
Name: "web-server-01",
Port: 8080,
Status: Running,
Active: true,

// Network settings
Config: {
    MaxConnections: 1000,
    Timeout: 30.5,
    KeepAlive: true,
},

// Tags for this server
Tags: ["production", "web", "primary"],

// Containers running on this server
Containers: [
    {
        Name: "nginx",
        Image: "nginx:latest",
        Ports: [80, 443],
    },
    {
        Name: "app",
        Image: "myapp:v1",
        Ports: [8080],
    },
],

// Environment labels
Labels: @map {
    "env": "production",
    "region": "us-west-2",
},

// SSL certificate (base64)
Certificate: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...",

// Description
Description: `This server handles
incoming web traffic for the
production environment.`,
```

## Go API

The clawtext package provides the following functions:

### Marshaling

```go
// Marshal to a pooled buffer (remember to release)
buf, err := clawtext.Marshal(ctx, myStruct)
defer buf.Release(ctx)

// Marshal directly to an io.Writer
err := clawtext.MarshalWriter(ctx, myStruct, writer)
```

### Unmarshaling

```go
// Unmarshal from bytes
var myStruct MyType
err := clawtext.Unmarshal(ctx, data, &myStruct)

// Unmarshal from an io.Reader
err := clawtext.UnmarshalReader(ctx, reader, &myStruct)
```

### Options

```go
// Use enum numbers instead of names
clawtext.Marshal(ctx, s, clawtext.WithUseEnumNumbers(true))

// Use hex encoding for bytes
clawtext.Marshal(ctx, s, clawtext.WithUseHexBytes(true))

// Custom indentation (default is 4 spaces)
clawtext.Marshal(ctx, s, clawtext.WithIndent("  "))

// Ignore unknown fields during unmarshal
clawtext.Unmarshal(ctx, data, &s, clawtext.WithIgnoreUnknownFields(true))
```

## YAML Pitfalls Avoided

Clawtext explicitly avoids common YAML issues:

| YAML Pitfall | Clawtext Solution |
|--------------|-------------------|
| `NO` interpreted as `false` | Strings always quoted, booleans only `true`/`false` |
| `yes`/`on` as boolean | Only `true` and `false` are booleans |
| `3.14` vs `"3.14"` ambiguity | Numbers are unquoted, strings are quoted |
| `012` as octal | Numbers are decimal, hex uses explicit `0x` prefix |
| Indentation-sensitive | Uses explicit delimiters `{}` and `[]` |
| Implicit type coercion | All types are explicit |
| Norway problem (`NO` -> `false`) | Enum names are unquoted identifiers |

## Comparison with Other Formats

| Feature | Clawtext | JSON | YAML |
|---------|----------|------|------|
| Comments | Yes | No | Yes |
| Trailing commas | Optional | No | N/A |
| Multi-line strings | Backticks | No | Yes |
| Explicit types | Yes | Partial | No |
| Human editable | Yes | Partial | Yes |
| Whitespace flexible | Yes | Yes | No |
