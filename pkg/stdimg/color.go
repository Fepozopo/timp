package stdimg

import (
	"image"
	"math"
)

// RGB<->HSL conversions operate on 0..1 floats.

func rgbToHsl(r, g, b float64) (h, s, l float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2
	if max == min {
		// achromatic
		h = 0
		s = 0
		return
	}
	d := max - min
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func hslToRgb(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		// achromatic
		r = l
		g = l
		b = l
		return
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r = hueToRgb(p, q, h+1.0/3.0)
	g = hueToRgb(p, q, h)
	b = hueToRgb(p, q, h-1.0/3.0)
	return
}

// Modulate adjusts brightness (percent), saturation (percent), and hue (degrees).
// brightness and saturation are given as percentages where 100 means unchanged.
// hue is in degrees and will be added to the hue channel.
func Modulate(src *image.NRGBA, brightnessPct, saturationPct, hueDegrees float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(b)
	bFactor := brightnessPct / 100.0
	sFactor := saturationPct / 100.0
	hueShift := hueDegrees / 360.0 // convert to 0..1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0]) / 255.0
			g := float64(src.Pix[i+1]) / 255.0
			b_ := float64(src.Pix[i+2]) / 255.0
			a := src.Pix[i+3]

			h, s, l := rgbToHsl(r, g, b_)
			// apply hue shift
			h = math.Mod(h+hueShift, 1.0)
			// adjust saturation and lightness
			s = clamp01(s * sFactor)
			l = clamp01(l * bFactor)
			r2, g2, b2 := hslToRgb(h, s, l)
			out.Pix[i+0] = uint8(clampFloatToUint8(r2 * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(g2 * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(b2 * 255.0))
			out.Pix[i+3] = a
		}
	}
	return out
}
