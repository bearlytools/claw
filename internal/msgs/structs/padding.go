package structs

var (
	padding1Bytes = make([]byte, 1)
	padding2Bytes = make([]byte, 2)
	padding3Bytes = make([]byte, 3)
	padding4Bytes = make([]byte, 4)
	padding5Bytes = make([]byte, 5)
	padding6Bytes = make([]byte, 6)
	padding7Bytes = make([]byte, 7)
)

// SizeWithPadding returns the complete size once padding has been applied.
func SizeWithPadding(size int) int {
	return size + int(8-(size%8))
}

// PaddingNeeded returns the amount of padding needed for size to align 64 bits.
func PaddingNeeded(size int) int {
	return int(8 - (size % 8))
}

// Padding returns a pre-allocated []byte that represents the padding we need to align
// to 64 bits.
func Padding(padding int) []byte {
	if padding > 7 {
		panic("ok buddy, we are 64 bit aligned, so why are you trying to pad more than 7 bytes?")
	}
	switch padding {
	case 0:
		return []byte{}
	case 1:
		return padding1Bytes
	case 2:
		return padding2Bytes
	case 3:
		return padding3Bytes
	case 4:
		return padding4Bytes
	case 5:
		return padding5Bytes
	case 6:
		return padding6Bytes
	case 7:
		return padding7Bytes
	}
	panic("should never get here")
}
