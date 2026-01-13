package stdimg

import (
	"image"
)

// SepiaTone applies a sepia color transform to src. Percentage is in 0..1
// where 1.0 is full sepia and 0.0 returns the original image.
func SepiaTone(src *image.NRGBA, percentage float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	if percentage <= 0 {
		return src
	}
	if percentage > 1 {
		percentage = 1
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			a := src.Pix[i+3]

			// Sepia matrix approximation operating on 0..255
			sr := 0.393*r + 0.769*g + 0.189*b_
			sg := 0.349*r + 0.686*g + 0.168*b_
			sb := 0.272*r + 0.534*g + 0.131*b_

			// clamp to 0..255
			if sr > 255 {
				sr = 255
			}
			if sg > 255 {
				sg = 255
			}
			if sb > 255 {
				sb = 255
			}

			// Blend with original by percentage
			rOut := (1.0-percentage)*r + percentage*sr
			gOut := (1.0-percentage)*g + percentage*sg
			bOut := (1.0-percentage)*b_ + percentage*sb

			out.Pix[i+0] = uint8(clampFloatToUint8(rOut))
			out.Pix[i+1] = uint8(clampFloatToUint8(gOut))
			out.Pix[i+2] = uint8(clampFloatToUint8(bOut))
			out.Pix[i+3] = a
		}
	}
	return out
}
