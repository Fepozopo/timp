package stdimg

import (
	"image"
	"math"
)

// Posterize reduces color levels per channel to `levels`.
func Posterize(src *image.NRGBA, levels int) *image.NRGBA {
	if src == nil {
		return nil
	}
	if levels < 2 {
		return CloneNRGBA(src)
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(b)
	step := 255.0 / float64(levels-1)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			a := src.Pix[i+3]
			rq := math.Round(r/step) * step
			gq := math.Round(g/step) * step
			bq := math.Round(b_/step) * step
			out.Pix[i+0] = uint8(clampFloatToUint8(rq))
			out.Pix[i+1] = uint8(clampFloatToUint8(gq))
			out.Pix[i+2] = uint8(clampFloatToUint8(bq))
			out.Pix[i+3] = a
		}
	}
	return out
}
