package stdimg

import (
	"image"
	"math"
)

// Vignette applies a radial darkening centered at (cx,cy).
// radius specifies the maximum radius (in pixels) where effect reaches full strength.
// sigma controls the smoothness (gaussian falloff). If radius<=0, it's set to half the image diagonal.
// strength controls maximum darkening where 1.0 means full darkening (mask effect fully applied),
// 0.0 means no effect. Values outside [0,1] are clamped.
// The implementation darkens pixels by multiplying RGB by (1 - mask*strength) where mask in [0,1].
// The mask is continuous: mask(d) = (1 - exp(-0.5*(d^2)/(sigma^2))) / (1 - exp(-0.5*(radius^2)/(sigma^2))).
// This yields mask(0)=0 and mask(radius)=1 (normalized), and clamps to [0,1] for d>radius.
func Vignette(src *image.NRGBA, radius, sigma float64, cx, cy int, strength float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if radius <= 0 {
		radius = math.Hypot(float64(w), float64(h)) / 2.0
	}
	if sigma <= 0 {
		sigma = radius / 3.0
	}
	// precompute normalizer at radius
	normAtRadius := 1 - math.Exp(-0.5*(radius*radius)/(sigma*sigma))
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			a := src.Pix[i+3]

			dx := float64(x - cx)
			dy := float64(y - cy)
			d := math.Hypot(dx, dy)

			// continuous normalized gaussian-like mask
			val := 1 - math.Exp(-0.5*(d*d)/(sigma*sigma))
			mask := val
			if normAtRadius > 0 {
				mask = val / normAtRadius
			}
			if mask < 0 {
				mask = 0
			}
			if mask > 1 {
				mask = 1
			}
			factor := 1.0 - mask
			out.Pix[i+0] = uint8(clampFloatToUint8(r * factor))
			out.Pix[i+1] = uint8(clampFloatToUint8(g * factor))
			out.Pix[i+2] = uint8(clampFloatToUint8(b_ * factor))
			out.Pix[i+3] = a
		}
	}
	return out
}
