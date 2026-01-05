package stdimg

import (
	"image"
	"math"
)

// AdaptiveSharpen uses UnsharpMask/Sharpen as an approximation for ImageMagick's
// adaptive sharpen. Parameters: radius (unused here), sigma (blur sigma) and amount.
func AdaptiveSharpen(src *image.NRGBA, radius, sigma, amount float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	// if radius==0 treat as auto: pick sigma based on image size when sigma <= 0
	if amount <= 0 {
		amount = 1.0
	}
	if radius == 0 || sigma <= 0 {
		// estimate sigma from image gradients: stronger edges -> smaller sigma
		b := src.Bounds()
		w := b.Dx()
		h := b.Dy()
		// compute average gradient magnitude using simple finite differences
		total := 0.0
		count := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				idx := src.PixOffset(x+b.Min.X, y+b.Min.Y)
				r := float64(src.Pix[idx+0])
				g := float64(src.Pix[idx+1])
				b_ := float64(src.Pix[idx+2])
				lum := 0.2126*r + 0.7152*g + 0.0722*b_
				// dx
				dx := 0.0
				if x+1 < w {
					idx2 := src.PixOffset(x+1+b.Min.X, y+b.Min.Y)
					r2 := float64(src.Pix[idx2+0])
					g2 := float64(src.Pix[idx2+1])
					b2 := float64(src.Pix[idx2+2])
					lum2 := 0.2126*r2 + 0.7152*g2 + 0.0722*b2
					dx = lum2 - lum
				}
				// dy
				dy := 0.0
				if y+1 < h {
					idx3 := src.PixOffset(x+b.Min.X, y+1+b.Min.Y)
					r3 := float64(src.Pix[idx3+0])
					g3 := float64(src.Pix[idx3+1])
					b3 := float64(src.Pix[idx3+2])
					lum3 := 0.2126*r3 + 0.7152*g3 + 0.0722*b3
					dy = lum3 - lum
				}
				mag := math.Abs(dx) + math.Abs(dy)
				total += mag
				count++
			}
		}
		meanGrad := total / float64(count+1)
		// heuristic mapping: sigma = clamp(16 / meanGrad, 0.5, 2.0)
		sigma = 16.0 / (meanGrad + 1e-6)
		if sigma < 0.5 {
			sigma = 0.5
		}
		if sigma > 2.0 {
			sigma = 2.0
		}
	}
	// UnsharpMask accepts radius, sigma, amount, threshold
	return UnsharpMask(src, radius, sigma, amount, 0.0)
}
