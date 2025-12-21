package conversions

import (
	"bytes"
	"testing"
)

func TestBytesToNumAndNumToBytes(t *testing.T) {
	data := []struct {
		T     string
		value []byte
		num   int
	}{
		{"uint8", []byte{1}, 1},
		{"int8", []byte{255}, -1},
		{"uint16", []byte{1, 1}, 257},
		{"int16", []byte{255, 254}, -257},
		{"uint32", []byte{3, 0, 1, 0}, 65539},
		{"int32", []byte{253, 255, 254, 255}, -65539},
		{"uint64", []byte{223, 94, 248, 255, 0, 0, 0, 0}, 4294467295},
		{"int64", []byte{33, 161, 7, 0, 255, 255, 255, 255}, -4294467295},
	}

	for _, d := range data {
		switch d.T {
		case "uint8":
			got := BytesToNum[uint8](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "int8":
			got := BytesToNum[int8](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "uint16":
			got := BytesToNum[uint16](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "int16":
			got := BytesToNum[int16](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "uint32":
			got := BytesToNum[uint32](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "int32":
			got := BytesToNum[int32](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "uint64":
			got := BytesToNum[uint64](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		case "int64":
			got := BytesToNum[int64](d.value)
			if int(*got) != d.num {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %d, want %d", d.T, *got, d.num)
			}
			b := NumToBytes(got)
			if !bytes.Equal(b, d.value) {
				t.Errorf("TestBytesToNumAndNumToBytes(%s): got %v, want %v", d.T, b, d.value)
			}
		}
	}
}
