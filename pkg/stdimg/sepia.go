package stdimg

import (
	"image"
	"image/color"
	"math"
)

// SepiaTone applies a sepia color transform to src using Lab-space blending.
// Percentage is in 0..1 where 1.0 is full sepia (target color) and 0.0 returns the original image.
// Target sepia color chosen: #704214 (a warm mid-brown).
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

	// Precompute target sepia Lab
	tcol, _ := parseHexColor("#704214")
	var tNRGBA color.NRGBA
	if c, ok := tcol.(color.NRGBA); ok {
		tNRGBA = c
	} else {
		r, g, b, a := tcol.RGBA()
		tNRGBA = color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	}
	trLin := srgbToLinear(tNRGBA.R)
	tgLin := srgbToLinear(tNRGBA.G)
	tbLin := srgbToLinear(tNRGBA.B)
	tX, tY, tZ := linearToXyz(trLin, tgLin, tbLin)
	Lsep, asep, bsep := xyzToLab(tX, tY, tZ)

	bounds := src.Bounds()
	out := image.NewNRGBA(bounds)
	w := bounds.Dx()
	h := bounds.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			alpha := src.Pix[i+3]
			// If fully transparent, keep as-is
			if alpha == 0 {
				out.Pix[i+0] = src.Pix[i+0]
				out.Pix[i+1] = src.Pix[i+1]
				out.Pix[i+2] = src.Pix[i+2]
				out.Pix[i+3] = alpha
				continue
			}
			r := src.Pix[i+0]
			g := src.Pix[i+1]
			b := src.Pix[i+2]

			rLin := srgbToLinear(r)
			gLin := srgbToLinear(g)
			bLin := srgbToLinear(b)
			X, Y, Z := linearToXyz(rLin, gLin, bLin)
			L, aCh, bCh := xyzToLab(X, Y, Z)

			// Blend in Lab space toward target sepia Lab
			L2 := (1.0-percentage)*L + percentage*Lsep
			a2 := (1.0-percentage)*aCh + percentage*asep
			b2 := (1.0-percentage)*bCh + percentage*bsep

			// Convert back to linear RGB
			x2, y2, z2 := labToXYZ(L2, a2, b2)
			r2Lin, g2Lin, b2Lin := xyzToLinearRGB(x2, y2, z2)
			// Gamma-encode back to sRGB 0..1
			rOut := linearToSrgb8(r2Lin)
			gOut := linearToSrgb8(g2Lin)
			bOut := linearToSrgb8(b2Lin)

			// clamp and write (linearToSrgb8 returns 0..1)
			out.Pix[i+0] = uint8(clampFloatToUint8(rOut * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(gOut * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(bOut * 255.0))
			out.Pix[i+3] = alpha
		}
	}
	return out
}

// linearToSrgb8 converts linear 0..1 to sRGB 0..1 (not scaled to 0..255)
func linearToSrgb8(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

// labToXYZ converts Lab to XYZ
func labToXYZ(L, a, b float64) (x, y, z float64) {
	fy := (L + 16) / 116.0
	fx := fy + a/500.0
	fz := fy - b/200.0
	Xn := 0.95047
	Yn := 1.00000
	Zn := 1.08883
	x = Xn * finvLab(fx)
	y = Yn * finvLab(fy)
	z = Zn * finvLab(fz)
	return
}

func finvLab(t float64) float64 {
	delta := 6.0 / 29.0
	if t > delta {
		return t * t * t
	}
	return 3 * delta * delta * (t - 4.0/29.0)
}

// xyzToLinearRGB converts XYZ to linear RGB (0..1)
func xyzToLinearRGB(x, y, z float64) (r, g, b float64) {
	r = 3.2404542*x - 1.5371385*y - 0.4985314*z
	g = -0.9692660*x + 1.8760108*y + 0.0415560*z
	b = 0.0556434*x - 0.2040259*y + 1.0572252*z
	// clamp to 0..1
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}
	if r > 1 {
		r = 1
	}
	if g > 1 {
		g = 1
	}
	if b > 1 {
		b = 1
	}
	return
}
