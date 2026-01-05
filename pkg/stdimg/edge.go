package stdimg

import (
	"image"
	"math"
)

// Edge applies a simple Sobel edge detector and returns a grayscale image representing edge magnitude.
// This improved version supports optional pre-blur (sigma), scale multiplier, thresholding, and binary output.
func EdgeEx(src *image.NRGBA, sigma, scale, threshold float64, binary bool) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var proc *image.NRGBA
	if sigma > 0 {
		proc = SeparableGaussianBlur(src, sigma)
	} else {
		proc = CloneNRGBA(src)
	}

	// Sobel kernels
	gx := [3][3]float64{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	gy := [3][3]float64{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	mag := make([]float64, w*h)
	maxMag := 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sumX := 0.0
			sumY := 0.0
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					ix := x + kx
					iy := y + ky
					if ix < 0 {
						ix = 0
					} else if ix >= w {
						ix = w - 1
					}
					if iy < 0 {
						iy = 0
					} else if iy >= h {
						iy = h - 1
					}
					c := samplePixelClamped(proc, ix, iy)
					r := float64(c.R) / 255.0
					g := float64(c.G) / 255.0
					b_ := float64(c.B) / 255.0
					lum := 0.2126*r + 0.7152*g + 0.0722*b_
					kxv := gx[ky+1][kx+1]
					kyv := gy[ky+1][kx+1]
					sumX += lum * kxv
					sumY += lum * kyv
				}
			}
			m := math.Sqrt(sumX*sumX + sumY*sumY)
			if scale > 0 {
				m *= scale
			}
			if m > maxMag {
				maxMag = m
			}
			mag[y*w+x] = m
		}
	}

	out := image.NewNRGBA(b)
	// normalize to [0,255]
	norm := 1.0
	if maxMag > 0 {
		norm = 1.0 / maxMag
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m := mag[y*w+x] * norm * 255.0
			val := clampFloatToUint8(m)
			if threshold > 0 {
				if binary {
					if m >= threshold {
						val = 255
					} else {
						val = 0
					}
				} else {
					// zero-out below threshold
					if m < threshold {
						val = 0
					}
				}
			}
			i := out.PixOffset(x, y)
			out.Pix[i+0] = uint8(val)
			out.Pix[i+1] = uint8(val)
			out.Pix[i+2] = uint8(val)
			out.Pix[i+3] = 255
		}
	}
	return out
}

// Edge is a compatibility wrapper using EdgeEx with defaults (no pre-blur, scale applied directly).
func Edge(src *image.NRGBA, scale float64) *image.NRGBA {
	return EdgeEx(src, 0.0, scale, 0.0, false)
}
