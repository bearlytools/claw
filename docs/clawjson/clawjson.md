# Claw JSON Format

The clawjson package provides JSON serialization for Claw data structures.

## Basic Usage

```go
// Marshal to JSON
data, err := clawjson.Marshal(ctx, myStruct)

// Unmarshal from JSON
err := clawjson.Unmarshal(ctx, data, &myStruct)
```

## Field Serialization

| Claw Type | JSON Type |
|-----------|-----------|
| bool | boolean |
| int8-int64, uint8-uint64 | number |
| float32, float64 | number |
| string | string |
| bytes | string (base64) |
| struct | object |
| enum | string (name) or number |
| lists | array |
| Any | object (special format) |

## Any Type Format

The `Any` type uses special fields prefixed with `@` to identify the type. The `@` prefix is not valid in Claw field names, so there is no collision risk.

### Known Type (registered)

When the type hash matches a registered type, the full struct fields are included:

```json
{
  "Name": "container",
  "Data": {
    "@type": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
    "@fieldType": "Inner",
    "ID": 12345,
    "Value": "test value"
  }
}
```

- `@type`: The 16-byte SHAKE128 hash as a hex string
- `@fieldType`: The human-readable type name
- Remaining fields: The actual struct field values

### Unknown Type

When the type is not registered, raw data is provided for forwarding:

```json
{
  "Name": "container",
  "Data": {
    "@type": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
    "@value": "SGVsbG8gV29ybGQ="
  }
}
```

- `@type`: The 16-byte SHAKE128 hash as a hex string
- `@value`: Base64-encoded raw bytes

### List of Any

```json
{
  "Items": [
    {
      "@type": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
      "@fieldType": "Inner",
      "ID": 1,
      "Value": "first"
    },
    {
      "@type": "a1b2c3d4e5f67890a1b2c3d4e5f67890",
      "@fieldType": "Inner",
      "ID": 2,
      "Value": "second"
    }
  ]
}
```

## Notes

- Fields prefixed with `@` are metadata, not actual struct fields
- When unmarshaling, the `@type` hash is used to look up the correct type
- If the type is not registered, the raw bytes can still be forwarded to another system
