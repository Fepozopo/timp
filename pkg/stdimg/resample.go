package stdimg

import (
	"image"
	"math"
)

// sampleBilinear samples src at floating coordinates (x,y) using bilinear interpolation.
func sampleBilinear(src *image.NRGBA, x, y float64) (r, g, b, a float64) {
	if src == nil {
		return
	}
	bx := src.Bounds()
	// translate to pixel space indices
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1

	c00 := samplePixelClamped(src, x0, y0)
	c10 := samplePixelClamped(src, x1, y0)
	c01 := samplePixelClamped(src, x0, y1)
	c11 := samplePixelClamped(src, x1, y1)

	xFrac := x - float64(x0)
	yFrac := y - float64(y0)

	// interpolate horizontally
	r0 := float64(c00.R)*(1-xFrac) + float64(c10.R)*xFrac
	r1 := float64(c01.R)*(1-xFrac) + float64(c11.R)*xFrac

	g0 := float64(c00.G)*(1-xFrac) + float64(c10.G)*xFrac
	g1 := float64(c01.G)*(1-xFrac) + float64(c11.G)*xFrac

	b0 := float64(c00.B)*(1-xFrac) + float64(c10.B)*xFrac
	b1 := float64(c01.B)*(1-xFrac) + float64(c11.B)*xFrac

	a0 := float64(c00.A)*(1-xFrac) + float64(c10.A)*xFrac
	a1 := float64(c01.A)*(1-xFrac) + float64(c11.A)*xFrac

	// interpolate vertically
	r = r0*(1-yFrac) + r1*yFrac
	g = g0*(1-yFrac) + g1*yFrac
	b = b0*(1-yFrac) + b1*yFrac
	a = a0*(1-yFrac) + a1*yFrac

	// clamp coordinates to image bounds - but sampling clamp already handled
	_ = bx
	return
}

// sinc helper
func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	x = math.Pi * x
	return math.Sin(x) / x
}

// lanczosKernel returns lanczos weight for distance x with parameter a.
func lanczosKernel(x, a float64) float64 {
	x = math.Abs(x)
	if x < 1e-12 {
		return 1
	}
	if x >= a {
		return 0
	}
	return sinc(x) * sinc(x/a)
}

// ResampleLanczos resamples src to dstW x dstH using Lanczos with window a (commonly 3).
func ResampleLanczos(src *image.NRGBA, dstW, dstH int, a float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	srcB := src.Bounds()
	srcW := srcB.Dx()
	srcH := srcB.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	if dstW == 0 || dstH == 0 {
		return dst
	}

	// scale factors
	xScale := float64(srcW) / float64(dstW)
	yScale := float64(srcH) / float64(dstH)

	// for each destination pixel, compute source coordinate and apply lanczos window.
	for y := 0; y < dstH; y++ {
		sy := (float64(y)+0.5)*yScale - 0.5
		for x := 0; x < dstW; x++ {
			sx := (float64(x)+0.5)*xScale - 0.5
			// accumulate
			sumR, sumG, sumB, sumA := 0.0, 0.0, 0.0, 0.0
			weightSum := 0.0
			// kernel extent
			xMin := int(math.Floor(sx - a + 1))
			xMax := int(math.Ceil(sx + a - 1))
			yMin := int(math.Floor(sy - a + 1))
			yMax := int(math.Ceil(sy + a - 1))
			for yi := yMin; yi <= yMax; yi++ {
				wy := lanczosKernel(float64(yi)-sy, a)
				for xi := xMin; xi <= xMax; xi++ {
					wx := lanczosKernel(float64(xi)-sx, a)
					w := wx * wy
					c := samplePixelClamped(src, xi, yi)
					sumR += float64(c.R) * w
					sumG += float64(c.G) * w
					sumB += float64(c.B) * w
					sumA += float64(c.A) * w
					weightSum += w
				}
			}
			if weightSum == 0 {
				weightSum = 1
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = uint8(clampFloatToUint8(sumR / weightSum))
			dst.Pix[i+1] = uint8(clampFloatToUint8(sumG / weightSum))
			dst.Pix[i+2] = uint8(clampFloatToUint8(sumB / weightSum))
			dst.Pix[i+3] = uint8(clampFloatToUint8(sumA / weightSum))
		}
	}
	return dst
}

// clampFloatToUint8 ensures v in [0,255]
func clampFloatToUint8(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
