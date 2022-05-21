package imghash

import (
	"image/color"
	"log"
)

func rgbToY(r, g, b int) uint8 {
	y := uint8((19595*int32(r) + 38470*int32(g) + 7471*int32(b) + 1<<15) >> 16) //24)
	//log.Printf("2) r0, g0, b0 = %d, %d, %d => y          = %d", r, g, b, y)
	return y
}

func rgbToYJPEG(r, g, b int) uint8 {
	y := uint8(0.2990*float32(r) + 0.5870*float32(g) + 0.1140*float32(b))
	return y
}

func runTest(r, g, b int) uint8 {
	r0, g0, b0 := uint8(r), uint8(g), uint8(b)
	y, cb, cr := color.RGBToYCbCr(r0, g0, b0)
	if cb == cr {
	}
	//r1, g1, b1 := color.YCbCrToRGB(y, cb, cr)
	//log.Printf("1) r0, g0, b0 = %d, %d, %d => y,  cb, cr = %d, %d, %d => r1, g1, b1 = %d, %d, %d\n", r0, g0, b0, y, cb, cr, r1, g1, b1)
	return y
}

// TODO: make these actual tests
func trgb() {
	for r := 0; r < 256; r += 7 {
		for g := 0; g < 256; g += 5 {
			for b := 0; b < 256; b += 3 {
				y1 := runTest(r, g, b)
				y2 := rgbToY(r, g, b)
				y3 := rgbToYJPEG(r, g, b)

				if y1 != y2 {
					log.Printf("Mismatched Y values for r, g, b = %d, %d, %d :: %d vs %d", r, g, b, y1, y2)
				}

				if y2 != y3 {
					log.Printf("Invalid Y JPEG value for r, g, b = %d, %d, %d :: %d vs %d ... %f", r, g, b, y2, y3, 0.2990*float32(r)+0.5870*float32(g)+0.1140*float32(b))
				}
			}
		}
	}
}
