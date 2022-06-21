package imghash

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
	"time"
)

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
				y1 := rgbToY(uint8(r), uint8(g), uint8(b))

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
				y1 := rgbToY(uint8(r), uint8(g), uint8(b))
				y2, _, _ := color.RGBToYCbCr(uint8(r), uint8(g), uint8(b))

				if y1 != y2 {
					t.Fatalf("Mismatched Y StdLib values for r, g, b = %d, %d, %d :: %d vs %d", r, g, b, y1, y2)
				}
			}
		}
	}
}

// Old method of doing the hash, this test was added to make sure all future iterations of the hash funciton comply with the original
func testold(img *image.RGBA) (vdhash uint64, hdhash uint64) {
	var col color.RGBA

	pixels := make([][]uint8, img.Rect.Dy())
	for y := range pixels {
		pixels[y] = make([]uint8, img.Rect.Dx())
		for x := range pixels[y] {
			col = img.RGBAAt(x, y)
			pixels[y][x] = rgbToY(col.R, col.G, col.B)
		}
	}

	// x and y are technically mislabeled, since the x component of a 2D array is the inner arrays
	var offset uint
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			offset = uint(x*8 + y)
			// Horizontal hash.
			if pixels[x][y] < pixels[x][y+1] {
				vdhash |= 1 << offset
			}

			// Vertical hash.
			if pixels[x][y] < pixels[x+1][y] {
				hdhash |= 1 << offset
			}
		}
	}
	return
}

func TestPixels(t *testing.T) {
	rand.Seed(time.Now().Unix())
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(rand.Intn(255)), G: uint8(rand.Intn(255)), B: uint8(rand.Intn(255)), A: 0xff})
		}
	}

	vh1, hh1 := testold(img)
	vh2, hh2, err := differenceHash(img)

	if vh1 != vh2 || hh1 != hh2 {
		t.Fatalf("\nold hash: %d %d\nnew hash: %d %d\nerror: %s\n", vh1, hh1, vh2, hh2, err)
	}
}
