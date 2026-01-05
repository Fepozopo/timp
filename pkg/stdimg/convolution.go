package stdimg

import (
	"image"
	"math"
	"sync"
)

// gaussianKernel1D generates a 1D Gaussian kernel with given sigma. Returns kernel and half-width radius.
func gaussianKernel1D(sigma float64) ([]float64, int) {
	if sigma <= 0 {
		return []float64{1.0}, 0
	}
	// choose radius ~ ceil(3*sigma)
	radius := int(math.Ceil(3 * sigma))
	sz := radius*2 + 1
	kern := make([]float64, sz)
	sum := 0.0
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-0.5 * (float64(i) * float64(i)) / (sigma * sigma))
		kern[i+radius] = v
		sum += v
	}
	// normalize
	for i := range kern {
		kern[i] /= sum
	}
	return kern, radius
}

// SeparableGaussianBlur applies a separable gaussian blur to src and returns a new *image.NRGBA
func SeparableGaussianBlur(src *image.NRGBA, sigma float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	kern, radius := gaussianKernel1D(sigma)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	// temporary buffer for horiz pass
	tmp := image.NewNRGBA(image.Rect(0, 0, w, h))
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))

	// horizontal pass
	var wg sync.WaitGroup
	for y := 0; y < h; y++ {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			for x := 0; x < w; x++ {
				sr, sg, sb, sa := 0.0, 0.0, 0.0, 0.0
				wsum := 0.0
				for k := -radius; k <= radius; k++ {
					ix := x + k
					// clamp
					if ix < 0 {
						ix = 0
					} else if ix >= w {
						ix = w - 1
					}
					c := samplePixelClamped(src, ix, y)
					wgt := kern[k+radius]
					sr += float64(c.R) * wgt
					sg += float64(c.G) * wgt
					sb += float64(c.B) * wgt
					sa += float64(c.A) * wgt
					wsum += wgt
				}
				i := tmp.PixOffset(x, y)
				tmp.Pix[i+0] = uint8(clampFloatToUint8(sr / wsum))
				tmp.Pix[i+1] = uint8(clampFloatToUint8(sg / wsum))
				tmp.Pix[i+2] = uint8(clampFloatToUint8(sb / wsum))
				tmp.Pix[i+3] = uint8(clampFloatToUint8(sa / wsum))
			}
		}(y)
	}
	wg.Wait()

	// vertical pass
	for x := 0; x < w; x++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			for y := 0; y < h; y++ {
				sr, sg, sb, sa := 0.0, 0.0, 0.0, 0.0
				wsum := 0.0
				for k := -radius; k <= radius; k++ {
					iy := y + k
					if iy < 0 {
						iy = 0
					} else if iy >= h {
						iy = h - 1
					}
					c := samplePixelClamped(tmp, x, iy)
					wgt := kern[k+radius]
					sr += float64(c.R) * wgt
					sg += float64(c.G) * wgt
					sb += float64(c.B) * wgt
					sa += float64(c.A) * wgt
					wsum += wgt
				}
				i := dst.PixOffset(x, y)
				dst.Pix[i+0] = uint8(clampFloatToUint8(sr / wsum))
				dst.Pix[i+1] = uint8(clampFloatToUint8(sg / wsum))
				dst.Pix[i+2] = uint8(clampFloatToUint8(sb / wsum))
				dst.Pix[i+3] = uint8(clampFloatToUint8(sa / wsum))
			}
		}(x)
	}
	wg.Wait()
	return dst
}
