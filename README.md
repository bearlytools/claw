# Claw

<p align="center">
  <img src="docs/claw_logo_light.svg" alt="Claw Logo" width="200">
</p>

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/bearlytools/claw?status.svg)](https://pkg.go.dev/github.com/bearlytools/claw)
[![Go Report Card](https://goreportcard.com/badge/github.com/bearlytools/claw)](https://goreportcard.com/report/github.com/bearlytools/claw)
![Go version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)

Since you've made it this far, why don't you hit that :star: up in the right corner.

<p align="center">
  <strong>A fast, zero-allocation binary serialization format with code generation</strong>
</p>

---

Claw is a binary serialization format and IDL (Interface Definition Language) designed for performance-critical applications. It prioritizes machine efficiency over wire size, offering lazy decoding and minimal heap allocations.

## Features

- **Lazy Decoding** - Only decode fields when accessed, significantly improving performance for large messages
- **Zero Heap Allocations** - Scalar type conversions require no heap allocations for languages that can support it
- **No Re-encoding** - Unchanged messages can be forwarded without re-encoding overhead
- **Message Patching** - Send and apply incremental patches instead of full messages
- **Built-in RPC** - Supports TCP, HTTP/1.1, HTTP/2, and Unix Domain Sockets
- **Reflection Support** - Runtime introspection of message structure
- **Set Detection** - Distinguish between unset fields and zero values via an option
- **Simple Tooling** - Single `clawc` binary handles all code generation

## Quick Start

### Installation

```bash
go install github.com/bearlytools/claw/clawc@latest
```

### Define a Schema

Create a `cars.claw` file:

```claw
package cars

import (
    "github.com/example/vins"
)

enum Maker uint8 {
    Unknown @0
    Toyota @1
    Ford @2
    Chevy @3
}

Struct Car {
    Name string @0
    Maker Maker @1
    Year uint16 @2
    Vin vins.Number @3
    PreviousVersions []Car @4
    Image bytes @5
}
```

### Setup claw.mod and claw.work files 


### Generate Code

```bash
cd <.claw file directory> && clawc
```

### Use in Go

```go
car := cars.NewCarFromRaw(
	cars.CarRaw{
		Name: "Chevelle",
		Maker: cars.Chevy,
		Year: 2024,
	}
)

// Encode
data := car.Marshal()

// Decode (lazy - fields decoded on access)
decoded, err := cars.UnmarshalCar(data)
if err != nil {
	// Do something
}
fmt.Println(decoded.Name())  // Fields decoded only when accessed
```

## Documentation

| Document | Description |
|----------|-------------|
| [Schema Language](docs/schema_language/schema_language.md) | Complete IDL syntax reference |
| [Modules](docs/modules/modules.md) | Package and import system |
| [Compilation](docs/compilation/compilation.md) | How `clawc` processes files |
| [Encoding](docs/encoding/encoding.md) | Binary wire format specification |
| [Replace](docs/replace/replace.md) | Local development with replace directives |

## Design Goals

- Machine-friendly binary format optimized for decode speed
- Zero heap allocations for data access in Go, Rust, and C++
- Single-pass encode/decode operations
- Support for Go, Rust, Zig, and JavaScript
- Intuitive import system (no `protoc` path gymnastics)
- Extensible serialization (JSON export support)
- Clear binary format specification

## Trade-offs

Claw optimizes for **speed over size**. The wire format uses fixed-size encodings and alignment padding rather than variable-length encoding. If minimizing bandwidth is your primary concern, Protocol Buffers or similar formats may be more appropriate.

## Comparison with Alternatives

### vs Protocol Buffers

Protocol Buffers is the industry standard with excellent language support and Google's backing. Choose Claw when:

- You need lazy decoding for large messages
- Heap allocations are a concern (GC pressure in high-throughput systems)
- You want simpler tooling than `protoc` + plugins
- You need to detect unset fields vs zero values without proto3 wrapper types

Choose Protocol Buffers when:

- You need broad language support
- You need enterprise support or compliance requirements
- Wire size is more important than decode speed
- You're already invested in the protobuf ecosystem

### vs Cap'n Proto

Cap'n Proto pioneered zero-copy serialization. Claw borrows the segment-based allocation strategy. Choose Claw when:

- You find Cap'n Proto's format difficult to implement
- You need an RPC system that's easier to port across languages
- You primarily work in Go

Choose Cap'n Proto when:

- You need the absolute fastest possible serialization
- You can use the C++ implementation or its bindings
- You need time-traveling RPC capabilities

### vs FlatBuffers

FlatBuffers offers zero-copy access patterns. Choose Claw when:

- You need better ergonomics for nested structures
- You want bounds checking and clear error handling
- You need consistent performance across different message shapes

## Current Status

- **Go**: Full support including RPC
- **Rust, Zig, JavaScript, Python**: Future plans

## Limited Benchmarks

## Contributing

Contributions are welcome. Please open an issue to discuss significant changes before submitting a PR.

## License

[MIT License](LICENSE) - Copyright (c) 2025 John G. Doak
