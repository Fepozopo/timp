package stdimg

import (
	"image"
	"math"
)

// UnsharpMask applies an unsharp mask to src. radius is unused (kept for API parity),
// sigma controls the gaussian blur, amount is multiplier, threshold is ignored if <=0.
func UnsharpMask(src *image.NRGBA, radius float64, sigma float64, amount float64, threshold float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	blurred := SeparableGaussianBlur(src, sigma)
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			sr := float64(src.Pix[i+0])
			sg := float64(src.Pix[i+1])
			sb := float64(src.Pix[i+2])
			sa := float64(src.Pix[i+3])

			bi := blurred.PixOffset(x, y)
			br := float64(blurred.Pix[bi+0])
			bg := float64(blurred.Pix[bi+1])
			bb := float64(blurred.Pix[bi+2])
			ba := float64(blurred.Pix[bi+3])

			// mask = src - blurred
			mr := sr - br
			mg := sg - bg
			mb := sb - bb

			if threshold > 0 {
				// threshold is in same units as ImageMagick (likely 0..QuantumRange) but here assume 0..255
				if math.Abs(mr) < threshold && math.Abs(mg) < threshold && math.Abs(mb) < threshold {
					// below threshold: copy original
					out.Pix[i+0] = uint8(clampFloatToUint8(sr))
					out.Pix[i+1] = uint8(clampFloatToUint8(sg))
					out.Pix[i+2] = uint8(clampFloatToUint8(sb))
					out.Pix[i+3] = uint8(clampFloatToUint8(sa))
					continue
				}
			}

			r := sr + amount*mr
			g := sg + amount*mg
			b_ := sb + amount*mb
			a_ := sa + amount*(sa-ba) // adjust alpha similarly

			out.Pix[i+0] = uint8(clampFloatToUint8(r))
			out.Pix[i+1] = uint8(clampFloatToUint8(g))
			out.Pix[i+2] = uint8(clampFloatToUint8(b_))
			out.Pix[i+3] = uint8(clampFloatToUint8(a_))
		}
	}
	return out
}

// Sharpen is a convenience wrapper using UnsharpMask. It accepts radius and sigma and uses amount=1.0
func Sharpen(src *image.NRGBA, radius float64, sigma float64) *image.NRGBA {
	return UnsharpMask(src, radius, sigma, 1.0, 0.0)
}

// Despeckle removes small speckles; simple wrapper around MedianFilter with a small radius.
func Despeckle(src *image.NRGBA, radius int) *image.NRGBA {
	if src == nil {
		return nil
	}
	if radius <= 0 {
		radius = 1
	}
	return MedianFilter(src, radius)
}
