<p align="center">
  <img src="docs/claw_logo_light.svg" alt="Claw Logo" width="200">
</p>

<p align="center">
    Claw
</p>

---

<p align="center">
  <strong>A fast, zero-allocation binary serialization format with code generation</strong>
</p>

---

<p align="center">
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  </a>
  <a href="https://pkg.go.dev/github.com/bearlytools/claw">
    <img src="https://godoc.org/github.com/bearlytools/claw?status.svg" alt="GoDoc">
  </a>
  <a href="https://goreportcard.com/report/github.com/bearlytools/claw">
    <img src="https://goreportcard.com/badge/github.com/bearlytools/claw" alt="Go Report Card">
  </a>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go" alt="Go version">
</p>

Claw is a binary serialization format and IDL (Interface Definition Language) designed for performance-critical applications. It prioritizes machine efficiency over wire size, offering lazy decoding and minimal heap allocations.

Since you've made it this far, why don't you hit that :star: up in the right corner.

## Features

- **Lazy Decoding** - Only decode fields when accessed, significantly improving performance for large messages
- **Almost Zero Heap Allocations** - Scalar type conversions require no heap allocations for languages that can support it and heavy reuse of buffers to avoid future allocations
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
- Close to Zero heap allocations for data access in Go (initial target language)
- Single-pass encode/decode operations
- No encoding for empty fields
- Eventual support for Rust, Zig, Pythobn and JavaScript
- Intuitive import system (no `protoc` path gymnastics)
- Extensible serialization (JSON export support to start with)
- Clear binary format specification

## Trade-offs

Claw optimizes for **speed over size**. The wire format uses fixed-size encodings and alignment padding rather than variable-length encoding. If minimizing bandwidth is your primary concern, Protocol Buffers or similar formats may be more appropriate.

## Comparison with Alternatives

**Note** These are micro-benchmarks using one message type. I cannot validate every type of marshal/unmarshal performance. Consider these rough benchmarks that tell part of the story, not the whole story. For almost any normal system this is just one minor part. Google has done quite well with proto2/3 for decades.

This does not cover cost of access (lazy decoding advantages) and many other details that will be covered in real world systems.

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

ProtoMarshal/ProtoUnmarshal benchmarks use the standard Go protocol buffer packages. I have not tried against various 3rd party enhancers like VTProto. This benchmark was for a Kubernetes Pod representation. The Patches represent using patch recording for 2 field changes in the Pod. This is a common thing for something like Kubernetes API server that will update a single date/time field and need to update all listeners.

Choose Protocol Buffers when:

- You need broad language support
- You need enterprise support or compliance requirements
- You're already invested in the protobuf ecosystem
- Wire size is more important than decode speed:

Quick size comparison:

 - Protocol Buffers: 14,416 bytes
 - Claw: 28,704 bytes
 - Patch (diff): 200 bytes (99.3% smaller)
 
Claw is twice the size on the wire, however this can be reduced with compression.  Patching reduces the size to trivial amounts if the changes remain small.

Consider using http://buf.build if you are using protocol buffers. Their Connect service is good and it eases dealing with protos.

### vs Cap'n Proto

Cap'n Proto pioneered zero-copy serialization. Claw borrows the segment-based allocation strategy with its own flavor. Choose Claw when:

- You find Cap'n Proto's format difficult to implement
- You want simple tooling
- Patching can help you
- You don't need the complexity of that RPC system, which mostly works for C++ with all features
- You primarily work in Go

Choose Cap'n Proto when:

- You need slightly faster de-serialization
- You can use the C++ implementation or its bindings
- You need time-traveling RPC capabilities
- Interfaces, groups and other features it provides are needed 
- You need a capabilities based system

| Benchmark           | ops/sec    | ns/op   | B/op    | allocs/op |
|---------------------|------------|---------|---------|-----------|
| CapnpMarshal        | 224,882    | 5,303   | 42,265  | 8         |
| CapnpUnmarshal      | 8,129,272  | 148     | 240     | 4         |
| ClawMarshal         | 313,525    | 3,761   | 32,768  | 1         |
| ClawUnmarshal       | 5,413,490  | 221     | 328     | 6         |
| ClawUnmarshalPooled | 4,671,649  | 254     | 218     | 4         |

Pooling is slower, but avoids allocations. `Cap'n Proto` can do pooling as well, but it greatly decreases performance in these benchmarks. I imagine overall it would add speed by avoid GC pauses.

### vs FlatBuffers

FlatBuffers offers zero-copy access patterns. Choose Claw when:

- You need better ergonomics for nested structures
- You want bounds checking and clear error handling
- You need consistent performance across different message shapes

Choose FlatBuffers when:

- Speed of decode needs to be almost free 

| Format           | Size         |
|------------------|--------------|
| Claw             | 28,704 bytes |
| FlatBuffers      | 29,312 bytes |

Marshal Performance

| Benchmark            | ns/op  | B/op   | allocs/op |
|----------------------|--------|--------|-----------|
| PatchMarshal         | 35     | 144    | 1         |
| ClawMarshal          | 3,866  | 32,768 | 1         |
| FlatbufMarshal       | 91,658 | 43,647 | 871       |
| FlatbufPooledMarshal | 86,423 | 10,469 | 864       |

Unmarshal Performance

| Benchmark            | ns/op   | B/op    | allocs/op |
|----------------------|---------|---------|-----------|
| FlatbufUnmarshal     | 0.36    | 0       | 0         |
| PatchUnmarshal       | 210     | 296     | 5         |
| ClawUnmarshal        | 218     | 328     | 6         |
| ClawUnmarshalPooled  | 255     | 218     | 4         |

Flatbuffer looses on marshalling, but unmarshal performance is unrivaled. So if unmarshalling is your only concern, Flatbuffers are significantly faster. 

Flatbuffer is made for games, but I think this probably only has benefits in C++ or Rust where a GC doesn't exist. GC costs and marshal performance look to kill any advantages in Go.

### vs JSON

JSON is the industry standard via REST. It is not a binary encoding, but included here as a comparison. While there is BSON, this is really a niche version.

Choose Claw when:

- You value any aspect of performance
- You are not required to implement JSON 
- You don't need a schema 

Choose JSON when:

- You are required to

Here is JSON performance information using the new Go json/v2 experimental package. This package greatly enhances Go's ability to deal with JSON at speed. But being a textual format, lacking information on message size and not doing lazy decode its perfomance is lackluster.

Marshal Performance

| Benchmark            | ns/op  | B/op    | allocs/op |
|----------------------|--------|---------|-----------|
| PatchMarshal         | 35     | 144     | 1         |
| ClawMarshal          | 3,866  | 32,768  | 1         |
| JSONMarshal          | 79,275  | 33,550 | 49        |

Unmarshal Performance

| Benchmark            | ns/op   | B/op    | allocs/op |
|----------------------|---------|---------|-----------|
| PatchUnmarshal       | 211     | 296     | 5         |
| ClawUnmarshal        | 222     | 328     | 6         |
| ClawUnmarshalPooled  | 257     | 218     | 4         |
| JSONUnmarshal        | 177,778 | 83,192  | 1,319     |

Serialized Sizes

| Format           | Bytes  |
|------------------|--------|
| Claw             | 28,704 |
| JSON             | 31,914 |

Some of the JSON stuff can be overcome. You can do a type of lazy decode if you know what you are getting aheady of time. You can also use file sizes and headers in transports to be smarter about allocations. You can use packages with compile time schemas as well. But at that point you are hacking away at something that is never going to be great at performance. JSON maximizes human readability and everything else is an after thought.  And I'd note, it probably would never have caught on if the world had not though XML was a great idea for a while.

## Current Status

- **Go**: Full support including RPC
- **Rust, Zig, JavaScript, Python**: Future plans (way, way, way in the future)

## Licenses

Code is:
[MIT License](LICENSE) - Copyright (c) 2025 John G. Doak

Claw®and Claw Format® are trademarks of John G. Doak.
The Claw logo is copyright John G. Doak.
