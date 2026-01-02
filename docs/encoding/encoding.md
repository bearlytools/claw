# Encoding Specification

### Alignment rule

Every value aligns to a 64bit word value (or 8 bytes). This means the smallest entry will be 8 bytes. It also means that a received Struct that is not byteSize % 8 == 0 is corrupted.

### Zero value rule

The zero value of a type are as follows:

| Field Type | Numeric Representation |
|------------|------------------------|
| integer    | 0                      |
| floats     | 0.0                    |
| bool       | false                  |
| string     | ""                     |
| bytes      | empty byte set or nil  |
| struct     | empty struct           |
| lists      | empty set or nil       |

Bytes, strings, struct and lists all translate slightly differently from language to language.
We will always allocate and set a value.

### Is set detection rule

All types other than lists can detect if the field was set or was simply the zero value. To do this requires the file option `IsSet()`. This causes an additiona byte or bytes to be encoded holding the set status.

It is safe to add it to an existing type, it is UNSAFE to remove it.

### Lists do not have IsSet rule

No list type has `IsSet()` variable detection. A list is only encoded if it has > 0 entries.

### Messages

Messages represent an encoded piece of data that will be unmarshalled into memory all at once (versus being streamed). A Message is a `root` level encoded Struct type containing all of its sub fields.

Message size must be completely determinable before writing. The first 8 bytes of an encoded Message will by the header for Struct field and the size stored in that header will be the size of the entire encoded message which will follow the alignment rule from above.

Messages should be able to be decoded in a single pass without ever moving back up the read buffer.

### The Generic Header

All fields use in Claw have a Generic header, which is defined as follows:

```
 ____________8 bytes_______________
|                                  |
+-----+-+--------------------------+
|  A  |B|            C             |   

A (2 bytes) Field number
B (1 byte) Field type
C (5 bytes) Data Portion
```

The `C` portion of the Generic Header is open for general use. Sometimes it defines the size of data that follows the header, sometimes it is the number of objects that follow.  And in some cases, it stores the value that is stored for that field type.

Field types are defined in this table:

| Field Type | Uint8 Representation   |
|------------|------------------------|
| unknown    | 0                      |
| bool       | 1                      |
| int8       | 2                      |
| int16      | 3                      |
| int32      | 4                      |
| int64      | 5                      |
| uint8      | 6                      |
| uint16     | 7                      |
| uint32     | 8                      |
| uint64     | 9                      |
| float32    | 10                     |
| float64    | 11                     |
| string     | 12                     |
| byte       | 13                     |
| struct     | 14                     |
| Any        | 15                     |
| []bool     | 41                     |
| []int8     | 42                     |
| []int16    | 43                     |
| []int32    | 44                     |
| []int64    | 45                     |
| []uint8    | 46                     |
| []uint16   | 47                     |
| []uint32   | 48                     |
| []uint64   | 49                     |
| []float32  | 50                     |
| []float64  | 51                     |
| []byte     | 52                     |
| []string   | 53                     |
| []struct   | 54                     |
| map        | 55                     |
| []Any      | 56                     |

## Encoding

### Structs

Struct encoding is made up of two sections:

```
+-------------------------------+
|  A  |           B             |   

Or with option:

+---------------------------------------------+
|  A  |           B             | C0 | C1 | ...


A (8 bytes) Generic Header
B (up to 1 Tebibyte) of field information
C IsSet detection entries. These are optional set by the IsSet() option. [1]byte entries, each bit reprsenting a field with 1 indicating it was set. The last bit indicates if there is another entry following 
this one.  
```

The field information section contains other encoded types that are decoded as they are reached. Each field in a Struct has a Generic Header, so once a decoder has moved its read index passed the Generic Header, it can read the next 8 bytes to get the field information in order to proceed with decoding.

The Generic Header for a Struct holds the size in bytes of the message. Because of its 40 bit size limit, the maximum size for a message is 1 Tebibyte.

The first Struct encountered (called the `root` Struct) must have its field number set to 0.

Structs that were set by a user are always encoded, regardless if they are in the zero value state.

A struct's size will always be divisble by 8 with a remainder of 0, otherwise it is malformed. All struct fields will be encoded as to be divisble by 8, or they are malformed.

### Bool

The `bool` type is completely encoded in the Generic Header. Simply the data portion has the first bit set to either 0 or 1 (false or true). The other 39 bits should be 0.

### Numeric types

All numeric types < 64 bits are encoded in the Generic Header in the data portion. The only difference is how it is encoded.

```
 ____________8 bytes_______________
|                                  |
+-----+-+--------------------------+
|  A  |B|            C             |   

A (2 bytes) Field number
B (1 byte) Field type
C (5 bytes) Encoded < 64 bit value
```

64 bit types have an additional 8 bytes following the header that contains the 64 bit encoded number. In this case, the 40 bits in the header are unused.

```
  16 bytes
 ___________
|           |
+-----------+
|  A  |  B  |

A (8 bytes) Generic Header with the last 40 bits set to 0
B (8 bytes) The encoded 64 bit value
```

Unsigned integers are encoded as is. Signed integers are bit shifted one position (<< 1) and if the value is negative, the value is the bitwise XOR'd of itself.

Floating points are encoded using the IEEE 754 binary representation.

### Bytes and String

Bytes and string are encoded the same way, the only difference being the difference in field type. It is up to the generated language package to ensure UTF-8 compliance for the string. From here on out, we will simply refer to bytes.

The bytes are encoded with the Generic Header, with the data portion being set to the number of bytes that are stored. The data itself is stored in a byte array equal to the number of bytes stored + padding to 64bits.  The header stores the size of the data portion without padding.

Padding is calculated with this pseudo code:

```
leftOver := bytesSize % 8
if leftOver == 0 {
    padding is 0
}
padding is 8 - (leftOver)
```

This requires the decoder to read the data size and calculate the data size + padding and read the entire value.  It can use the data size that was decoded to return only the relevant portion without the padding.

### Lists

No matter what the list, the list items should be encoded in the order that they are in the list.

#### List of bools

A list of bools is made up of a Generic Header and multiple of 8 bytes. The Generic Header's data portion is set to the number of bool values that are stored in the list. This gives a maximum list size of 2^40 or 1099511627776 entries. 

The data portion is always some multiple of 8 bytes (64 bits). Each 64 bits allocated can store up to 64 bools. From 1 to 64 bools requires 8 bytes, 65-128 requires 16 bytes, so on and so forth. 

```
 8 + X bytes
 ___________
|           |
+-----------+
|  A  |  B  |

A (8 bytes) Generic Header with the last 40 bits set the number of items in the list
B (X bytes) A number of bytes divisible by 8 with a remainder of 0 that stores bools
```

#### Lists of numbers

A list of numbers includes any numeric type. It is made up of the Generic Header and some multiple of 8 bytes (64 bits). The Generic Header's data portion is set to the number of items in the list. This gives a maximum list size of 2^40 or 1099511627776 entries. 

After the header, the number of words (8 bytes) allocated is the number of words that is required to hold the length of the list * size of the numeric type. So if we wanted to store 8 bit numbers (uint8 or int8) and wanted to store up to 4 of them, a single word (8 bytes) would be allocated. If we appended to that list to add a 5th value, two words would be required (and would hold until we needed to store a 9th value).

```
 8 + X bytes
 ___________
|           |
+-----------+
|  A  |  B  |

A (8 bytes) Generic Header with the last 40 bits set the number of items in the list
B (X bytes) A number of bytes divisible by 8 with a remainder of 0 that stores numbers of some size
```

#### Lists of strings or bytes

Similar to either the string or bytes type from above, a list of strings or bytes is encoded the same way with only the field type different. We will refer only to list of bytes for rest of this section.

A list of bytes is a Generic Header with the data portion set to the number of items in the list and a set of list entries. This gives a maximum list size of 2^40 or 1099511627776 entries. 

A list entry is made up of a 4 byte header that holds the size in bytes that the entry will be and that entry in bytes. This gives a maximum entry size of 4096 Mebibyte or a little over 4 GiB.

```
+-----------+---------+----------------------+---------+----------------------+---------+
|     A     |    B    |          C           |    B    |          C           |    D    |

A (8 bytes)   Generic Header with the last 40 bits set to the number of items in the list
B (4 bytes)   A 32 bit number represening the size of an entry in bytes
C (X bytes)   The entry data
D (0-7 bytes) Padding to round out the list to be divisible by 8 bytes
```

#### Lists of structs

List of structs are encoded with a Generic Header with the data portion set to the number of items in the list and the encoded structs. All structs must be the same type, as denoted by its name.

```
+-------+-----------+-----------+-----------+
|   A   |     B     |     B     |     B     |

A (8 bytes)   Generic Header with the last 40 bits set the number of items in the list
B (X bytes)   An encoded struct type

When encoding lists of Structs, the Struct items have their field number set to the index number in the array.

### Any

The Any type stores a value of any Claw struct type along with a type identifier hash. This allows polymorphic message handling where the concrete type is determined at runtime.

The Generic Header's data portion (40 bits) is structured as:
- Bits 0-7: The real underlying type (always 14 for struct currently)
- Bits 8-39: The data size in bytes (32 bits, up to 4GB)

```
 ____________8 bytes______________ + X bytes
|                                 |
+-----+-+-------------------------+------------------+------------------+-------+
|  A  |B|           C             |        D         |        E         |   F   |

A (2 bytes)  Field number
B (1 byte)   Field type (15 for Any)
C (5 bytes)  Data portion: bits 0-7 = real type (14), bits 8-39 = data size
D (16 bytes) SHAKE128 hash of full type path (e.g., "github.com/pkg.TypeName")
E (X bytes)  Serialized struct data
F (0-7 bytes) Padding to 64-bit alignment
```

The type hash is computed from the full import path concatenated with the type name (e.g., `github.com/example/pkg.MyStruct`). This provides a unique identifier for each type across packages.

#### List of Any

A list of Any values uses the same header format as other lists, with each item containing its own type hash and data.

```
+-------+------------------+------------------+------------------+------------------+-------+
|   A   |        B1        |        C1        |        B2        |        C2        |   D   |

A (8 bytes)   Generic Header with field type 55, data portion = item count
B (16 bytes)  SHAKE128 hash for item N
C (X bytes)   Serialized struct data for item N (size determined by reading struct header)
D (0-7 bytes) Final padding to 64-bit alignment
```

Each item in the list is self-describing: the hash identifies the type, and the struct header contains the size of that item's data. This allows heterogeneous lists where each item can be a different concrete type.

### Maps

Maps provide key-value storage with typed keys and values. Maps are encoded with a specialized header that includes both key and value type information.

#### Map Header

The map field uses a Generic Header with the data portion structured as follows:

```
 ____________8 bytes_______________
|                                  |
+-----+-+--------------------------+
|  A  |B|            C             |

A (2 bytes)  Field number
B (1 byte)   Field type (55 for map)
C (5 bytes)  Data portion:
             - Bits 0-7:   Key type (uint8)
             - Bits 8-15:  Value type (uint8)
             - Bits 16-39: Total size in bytes (24 bits, max 16 MiB)
```

The key type and value type use the same numeric representation as the field type table above.

#### Map Data

After the header, map entries are encoded sequentially without individual padding:

```
+-------+---------+---------+---------+---------+-------+
|   A   |   B1    |   C1    |   B2    |   C2    |   D   |

A (8 bytes)   Map Header
B (X bytes)   Encoded key N
C (X bytes)   Encoded value N
D (0-7 bytes) Padding to 64-bit alignment at the end
```

Keys are maintained in sorted order for deterministic encoding.

#### Key Encoding

Keys are encoded based on their type:

| Key Type | Encoding |
|----------|----------|
| bool     | 1 byte (0 or 1) |
| int8/uint8 | 1 byte |
| int16/uint16 | 2 bytes, little-endian |
| int32/uint32/float32 | 4 bytes, little-endian |
| int64/uint64/float64 | 8 bytes, little-endian |
| string   | 4-byte length (little-endian) + UTF-8 data |

Signed integers use the same zigzag encoding as regular fields. Floats use IEEE 754 binary representation.

#### Value Encoding

Values are encoded based on their type:

| Value Type | Encoding |
|------------|----------|
| bool       | 1 byte (0 or 1) |
| int8/uint8 | 1 byte |
| int16/uint16 | 2 bytes, little-endian |
| int32/uint32/float32 | 4 bytes, little-endian |
| int64/uint64/float64 | 8 bytes, little-endian |
| string     | 4-byte length (little-endian) + UTF-8 data |
| bytes      | 4-byte length (little-endian) + byte data |
| struct     | Full encoded struct (includes struct header) |
| map        | Full encoded map (nested, includes map header) |
| Any        | 16-byte SHAKE128 hash + encoded struct data |
| []Any      | 4-byte count + (16-byte hash + encoded struct) for each item |

#### Map with Any Values

For `map[K]Any`, each value is encoded as:

```
+------------------+------------------+
|        A         |        B         |

A (16 bytes) SHAKE128 hash of full type path (e.g., "github.com/pkg.TypeName")
B (X bytes)  Serialized struct data (size from embedded struct header)
```

#### Map with []Any Values

For `map[K][]Any`, each value is encoded as:

```
+--------+------------------+------------------+------------------+------------------+
|   A    |        B1        |        C1        |        B2        |        C2        |

A (4 bytes)   Item count (little-endian)
B (16 bytes)  SHAKE128 hash for item N
C (X bytes)   Serialized struct data for item N
```

Each item in the list can be a different concrete struct type, identified by its type hash.
