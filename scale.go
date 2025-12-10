package therm

import (
	"image"
	"image/color"

	"github.com/nfnt/resize"
)

// ScaleBicubicNFNT scales a 2D float64 array using nfnt/resize bicubic interpolation.
// scale = 2 => 2x width & 2x height
func ScaleBicubicNFNT(src [][]float64, scale int) [][]float64 {
	h := len(src)
	w := len(src[0])

	newW := uint(w * scale)
	newH := uint(h * scale)

	// Convert float64 2D array → image.Gray
	img := floatsToGray(src)

	// Use nfnt/resize (bicubic)
	resized := resize.Resize(newW, newH, img, resize.Bicubic)

	// Convert back to [][]float64
	return grayToFloats(resized.(*image.Gray))
}

// float64 → *image.Gray
func floatsToGray(src [][]float64) *image.Gray {
	h := len(src)
	w := len(src[0])

	img := image.NewGray(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(src[y][x]) // assumes 0–255 range; adjust if needed
			img.SetGray(x, y, color.Gray{Y: v})
		}
	}

	return img
}

// *image.Gray → [][]float64
func grayToFloats(img *image.Gray) [][]float64 {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	out := make([][]float64, h)
	for y := 0; y < h; y++ {
		row := make([]float64, w)
		for x := 0; x < w; x++ {
			row[x] = float64(img.GrayAt(x, y).Y)
		}
		out[y] = row
	}

	return out
}
