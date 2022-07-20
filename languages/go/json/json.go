// Package JSON provides for encoding and decoding Claw Structs into JSON.
package json

// Marshal will marshal a Claw Struct into the JSON representation. However, you should
// consider using Options.Write() instead and use a bytes.Buffer{} or strings.Builder to allow resuse
// and reduce allocations.
/*
func Marshal(s reflect.ClawStruct) ([]byte, error) {
	b := bytes.Buffer{}

	_, err := Write(&b)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
*/
