package structs

import "fmt"

// TODO(jdoak): Delete this?
var (
	padding1Bytes  = make([]byte, 1)
	padding2Bytes  = make([]byte, 2)
	padding3Bytes  = make([]byte, 3)
	padding4Bytes  = make([]byte, 4)
	padding5Bytes  = make([]byte, 5)
	padding6Bytes  = make([]byte, 6)
	padding7Bytes  = make([]byte, 7)
	padding8Bytes  = make([]byte, 8)
	padding9Bytes  = make([]byte, 9)
	padding10Bytes = make([]byte, 10)
	padding11Bytes = make([]byte, 11)
	padding12Bytes = make([]byte, 12)
	padding13Bytes = make([]byte, 13)
	padding14Bytes = make([]byte, 14)
	padding15Bytes = make([]byte, 15)
	padding16Bytes = make([]byte, 16)
	padding17Bytes = make([]byte, 17)
	padding18Bytes = make([]byte, 18)
	padding19Bytes = make([]byte, 19)
	padding20Bytes = make([]byte, 20)
	padding21Bytes = make([]byte, 21)
	padding22Bytes = make([]byte, 22)
	padding23Bytes = make([]byte, 23)
	padding24Bytes = make([]byte, 24)
	padding25Bytes = make([]byte, 25)
	padding26Bytes = make([]byte, 26)
	padding27Bytes = make([]byte, 27)
	padding28Bytes = make([]byte, 28)
	padding29Bytes = make([]byte, 29)
	padding30Bytes = make([]byte, 30)
	padding31Bytes = make([]byte, 31)
	padding32Bytes = make([]byte, 32)
	padding33Bytes = make([]byte, 33)
	padding34Bytes = make([]byte, 34)
	padding35Bytes = make([]byte, 35)
	padding36Bytes = make([]byte, 36)
	padding37Bytes = make([]byte, 37)
	padding38Bytes = make([]byte, 38)
	padding39Bytes = make([]byte, 39)
	padding40Bytes = make([]byte, 40)
	padding41Bytes = make([]byte, 41)
	padding42Bytes = make([]byte, 42)
	padding43Bytes = make([]byte, 43)
	padding44Bytes = make([]byte, 44)
	padding45Bytes = make([]byte, 45)
	padding46Bytes = make([]byte, 46)
	padding47Bytes = make([]byte, 47)
	padding48Bytes = make([]byte, 48)
	padding49Bytes = make([]byte, 49)
	padding50Bytes = make([]byte, 50)
	padding51Bytes = make([]byte, 51)
	padding52Bytes = make([]byte, 52)
	padding53Bytes = make([]byte, 53)
	padding54Bytes = make([]byte, 54)
	padding55Bytes = make([]byte, 55)
	padding56Bytes = make([]byte, 56)
	padding57Bytes = make([]byte, 57)
	padding58Bytes = make([]byte, 58)
	padding59Bytes = make([]byte, 59)
	padding60Bytes = make([]byte, 60)
	padding61Bytes = make([]byte, 61)
	padding62Bytes = make([]byte, 62)
	padding63Bytes = make([]byte, 63)
)

// Padding returns a pre-allocated []byte that represents the padding we need to align
// to 64 bits.
func Padding(padding uint8) []byte {
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
	case 8:
		return padding8Bytes
	case 9:
		return padding9Bytes
	case 10:
		return padding10Bytes
	case 11:
		return padding11Bytes
	case 12:
		return padding12Bytes
	case 13:
		return padding13Bytes
	case 14:
		return padding14Bytes
	case 15:
		return padding15Bytes
	case 16:
		return padding16Bytes
	case 17:
		return padding17Bytes
	case 18:
		return padding18Bytes
	case 19:
		return padding19Bytes
	case 20:
		return padding20Bytes
	case 21:
		return padding21Bytes
	case 22:
		return padding22Bytes
	case 23:
		return padding23Bytes
	case 24:
		return padding24Bytes
	case 25:
		return padding25Bytes
	case 26:
		return padding26Bytes
	case 27:
		return padding27Bytes
	case 28:
		return padding28Bytes
	case 29:
		return padding29Bytes
	case 30:
		return padding30Bytes
	case 31:
		return padding31Bytes
	case 32:
		return padding32Bytes
	case 33:
		return padding33Bytes
	case 34:
		return padding34Bytes
	case 35:
		return padding35Bytes
	case 36:
		return padding36Bytes
	case 37:
		return padding37Bytes
	case 38:
		return padding38Bytes
	case 39:
		return padding39Bytes
	case 40:
		return padding40Bytes
	case 41:
		return padding41Bytes
	case 42:
		return padding42Bytes
	case 43:
		return padding43Bytes
	case 44:
		return padding44Bytes
	case 45:
		return padding45Bytes
	case 46:
		return padding46Bytes
	case 47:
		return padding47Bytes
	case 48:
		return padding48Bytes
	case 49:
		return padding49Bytes
	case 50:
		return padding50Bytes
	case 51:
		return padding51Bytes
	case 52:
		return padding52Bytes
	case 53:
		return padding53Bytes
	case 54:
		return padding54Bytes
	case 55:
		return padding55Bytes
	case 56:
		return padding56Bytes
	case 57:
		return padding57Bytes
	case 58:
		return padding58Bytes
	case 59:
		return padding59Bytes
	case 60:
		return padding60Bytes
	case 61:
		return padding61Bytes
	case 62:
		return padding62Bytes
	case 63:
		return padding63Bytes
	}
	panic(fmt.Sprintf("cannot get 64 bit padding for %d", padding))
}
