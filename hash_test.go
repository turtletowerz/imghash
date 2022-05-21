package imghash

import (
	"image/color"
	"testing"
)

// TODO: combine this with the one in hash.go so that test can use that and does not need to be updated separately
func rgbToY(r, g, b int) uint8 {
	return uint8((19595*int32(r) + 38470*int32(g) + 7471*int32(b) + 1<<15) >> 16)
}

// Subtract two numbers and return 0 or a whole number
func subAbs(n1, n2 uint8) uint8 {
	if n1 > n2 {
		return n1 - n2
	}
	return n2 - n1
}

func TestHashJPEG(t *testing.T) {
	for r := 0; r < 256; r += 7 {
		for g := 0; g < 256; g += 5 {
			for b := 0; b < 256; b += 3 {
				y1 := rgbToY(r, g, b)

				// Implementation according to the JFIF
				yfloat := 0.2990*float32(r) + 0.5870*float32(g) + 0.1140*float32(b)
				y2 := uint8(yfloat)

				// We allow the hashes to be 1 off because number such as 16.9 are floored to 16 in the JPEG impl, but ceiled to 17 in the builtin one.
				if y1 != y2 && subAbs(y1, y2) > 1 {
					t.Fatalf("Mismatched Y JPEG value for r, g, b = %d, %d, %d :: %d vs %d => %f", r, g, b, y1, y2, yfloat)
				}
			}
		}
	}
}

func TestHashStdLib(t *testing.T) {
	for r := 0; r < 256; r += 7 {
		for g := 0; g < 256; g += 5 {
			for b := 0; b < 256; b += 3 {
				y1 := rgbToY(r, g, b)
				y2, _, _ := color.RGBToYCbCr(uint8(r), uint8(g), uint8(b))

				if y1 != y2 {
					t.Fatalf("Mismatched Y StdLib values for r, g, b = %d, %d, %d :: %d vs %d", r, g, b, y1, y2)
				}
			}
		}
	}
}
