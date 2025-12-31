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
- **Built-in RPC** - Supports TCP, HTTP/1.1, HTTP/2 and Unix Domain Sockets
- **Reflection Support** - Runtime introspection of message structure
- **Set Detection** - Distinguish between unset fields and zero values via a low cost option
- **Simple Tooling** - Single `clawc` binary handles all code generation

## Quick Start

### Installation

```bash
go install github.com/bearlytools/claw/clawc@latest
```

This installs the `clawc` compiler which is the only tool you need to use `Claw`.

### Define a Schema

Create a `cars.claw` file:

```claw
package github.com/[your repo]/vehicles/cars

import (
    "github.com/[your repo]/vehicles/vins"
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

Claw requires a `claw.mod` file located in the directory with every `.claw` file. At the top of your `git` repo or the root directory where `.claw` files will be located if not in a `git` repo, we will need to add a `claw.work` file.

Loosely based on `go.mod` and `go.work` files, these allow us to know where our local dependencies exist and where to fetch our external dependencies. 

**Note** We currently only support `github` for external `git` repositories. Future work will untether us from `github` and add support for other `VCS` systems.

#### Generate the claw.mod file 

```bash
cd [directory containing claw file]
clawc mod init [github.com/repo/path]
```

#### Generate the claw.work file 

```bash 
clawc work init [github.com/path/]
```

This will include a line in the file:
```
vendorDir claw_vendor
```

This is where all 3rd party claw files will be downloaded and stored, relative to the root. If you want this to be another directory, change this to the path you want. 

**Note** It is not a good idea to use `vendor` as that tends to interfere with other things such as Go's `vendor` directory support.


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
- You need a lot more speed:

```
| Benchmark               | ops/sec    | ns/op   | B/op    | allocs/op |
| ProtoMarshal            | 22,720     | 52,196  | 21,440  | 317       |
| ProtoUnmarshal          | 10,000     | 103,697 | 101,169 | 2,355     |
| ClawMarshal             | 313,525    | 3,761   | 32,768  | 1         |
| ClawUnmarshal           | 5,413,490  | 221     | 328     | 6         |
| ClawUnmarshalPooled     | 4,671,649  | 254     | 218     | 4         |
| ClawPatchMarshal        | 33,352,981 | 36      | 144     | 1         |
| ClawPatchUnmarshal      | 5,384,586  | 216     | 296     | 5         |
```

ProtoMarshal/ProtoUnmarshal are the standard Go protocol buffer packages. I have not tried against various 3rd party enhancers like VTProto. This benchmark was for a Kubernetes Pod representation. The Patches represent using patch recording for 2 field changes in the Pod. This is a common thing for something like Kubernetes API server that will update a single date/time field and need to update all listeners.

Choose Protocol Buffers when:

- You need broad language support
- You need enterprise support or compliance requirements
- You're already invested in the protobuf ecosystem
- Wire size is more important than decode speed:

Here is that Pod on the wire:

 - Protocol Buffers: 14,416 bytes
 - Claw: 28,704 bytes
 - Patch (diff): 200 bytes (99.3% smaller)
 
We are twice the size on the wire, however this can be reduced with compression.  Patching reduces the size to trivial amounts if the changes remain small.

### vs Cap'n Proto

Cap'n Proto pioneered zero-copy serialization. Claw borrows the segment-based allocation strategy with its own flavort. Choose Claw when:

- You find Cap'n Proto's format difficult to implement
- You want simple tooling
- Patching can help you
- You don't need the complexity of that RPC system, which mostly works for C++ with all features
- You primarily work in Go

Choose Cap'n Proto when:

- You need the absolute fastest possible de-serialization
- You can use the C++ implementation or its bindings
- You need time-traveling RPC capabilities
- Interfaces, groups and other features are it provides are needed 
- You need a capabilities based system

| Benchmark           | ops/sec    | ns/op   | B/op    | allocs/op |
|---------------------|------------|---------|---------|-----------|
| CapnpMarshal        | 224,882    | 5,303   | 42,265  | 8         |
| CapnpUnmarshal      | 8,129,272  | 148     | 240     | 4         |
| ClawMarshal         | 313,525    | 3,761   | 32,768  | 1         |
| ClawUnmarshal       | 5,413,490  | 221     | 328     | 6         |
| ClawUnmarshalPooled | 4,671,649  | 254     | 218     | 4         |

Pooling is slower, but avoid allocations. `Cap'n Proto` can do pooling as well, but it greatly decreases performance in these benchmarks. I imagine overall it also adds to speed by avoid GC pauses.

### vs FlatBuffers

FlatBuffers offers zero-copy access patterns. Choose Claw when:

- You need better ergonomics for nested structures
- You want bounds checking and clear error handling
- You need consistent performance across different message shapes

## Current Status

- **Go**: Full support including RPC
- **Rust, Zig, JavaScript, Python**: Future plans

## Contributing

Contributions are welcome. Please open an issue to discuss significant changes before submitting a PR.

## License

[MIT License](LICENSE) - Copyright (c) 2025 John G. Doak
