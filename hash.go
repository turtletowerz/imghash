package imghash

import (
	"image"
	"image/color"
	"math/bits"

	"github.com/pkg/errors"
)

type Hash struct {
	VHash uint64
	HHash uint64

	// TODO: an ID? a Hash? what do we use to map these hashes to concrete output.
	Index uint32
}

// Returns the hamming distance between the two vertical hashes + hamming distance between the two horizontal hashes
func (i Hash) Distance(o Hash) int {
	return bits.OnesCount64(i.VHash^o.VHash) + bits.OnesCount64(i.HHash^o.HHash)
}

// From color.RGBToYCbCr in Go's standard library, but don't use RGBA() since RGBToYCbCr expects uint8s.
// It's a pretty ingenious solution that I can't seem to find in any other library.
// Upon testing it appears it performs exactly the same as the standard float
// conversion outlined by JPEG (https://www.w3.org/Graphics/JPEG/jfif3.pdf).
// The one exception is when the float is the upper half of a number (13.6, 15.8, 60.9)
// it will ceil so the results will respectively be (14, 16, 61). This provides a small
// amount of change, but it's so unnoticeable in terms of the hash that it's not worth using floats.
//
// This function is separate so it can be used for testing, but is inline-able
func rgbToY(r, g, b uint8) uint8 {
	return uint8((19595*int32(r) + 38470*int32(g) + 7471*int32(b) + 1<<15) >> 16)
}

// http://www.hackerfactor.com/blog/?/archives/529-Kind-of-Like-That.html
func differenceHash(img *image.NRGBA) (hdhash, vdhash uint64, err error) {
	// Check to make sure the bounds are the right size for the hash.
	dx, dy := img.Rect.Dx(), img.Rect.Dy()
	if dx != width || dy != height {
		err = errors.Errorf("Invalid dimensions %dx%d, must be a 9x9 image", dx, dy)
		return
	}

	var col color.NRGBA

	pixels := make([][]uint8, dy)
	for y := range pixels {
		pixels[y] = make([]uint8, dx)
		for x := range pixels[y] {
			col = img.NRGBAAt(x, y)
			pixels[y][x] = rgbToY(col.R, col.G, col.B)
		}
	}

	// Whether you do < or > for the comparison doesn't matter, it just has to be consistent.
	var offset uint64 = 1
	for y := 0; y < dy-1; y++ {
		for x := 0; x < dx-1; x++ {
			// Vertical hash.
			if pixels[y][x] < pixels[y+1][x] {
				vdhash |= offset
			}

			// Horizontal hash.
			if pixels[y][x] < pixels[y][x+1] {
				hdhash |= offset
			}

			offset <<= 1
		}
	}

	return
}
