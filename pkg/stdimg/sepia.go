package stdimg

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
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
	initSepiaLUTs()
	trLin := srgb8ToLinearLUT(tNRGBA.R)
	tgLin := srgb8ToLinearLUT(tNRGBA.G)
	tbLin := srgb8ToLinearLUT(tNRGBA.B)
	tX, tY, tZ := linearToXyz(trLin, tgLin, tbLin)
	Lsep, asep, bsep := xyzToLab(tX, tY, tZ)

	bounds := src.Bounds()
	out := image.NewNRGBA(bounds)
	w := bounds.Dx()
	h := bounds.Dy()
	// choose worker count
	workers := runtime.GOMAXPROCS(0)
	// fallback to single-threaded for small images
	if h < 64 || workers <= 1 {
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

				rLin := srgb8ToLinearLUT(r)
				gLin := srgb8ToLinearLUT(g)
				bLin := srgb8ToLinearLUT(b)
				X, Y, Z := linearToXyz(rLin, gLin, bLin)
				L, aCh, bCh := xyzToLab(X, Y, Z)

				// Blend in Lab space toward target sepia Lab
				L2 := (1.0-percentage)*L + percentage*Lsep
				a2 := (1.0-percentage)*aCh + percentage*asep
				b2 := (1.0-percentage)*bCh + percentage*bsep

				// Convert back to linear RGB
				x2, y2, z2 := labToXYZ(L2, a2, b2)
				r2Lin, g2Lin, b2Lin := xyzToLinearRGB(x2, y2, z2)
				// Gamma-encode back to sRGB 0..1 using LUT-accelerated approx
				rOut := linearToSrgbApprox(r2Lin)
				gOut := linearToSrgbApprox(g2Lin)
				bOut := linearToSrgbApprox(b2Lin)

				// clamp and write (linearToSrgbApprox returns 0..1)
				out.Pix[i+0] = uint8(clampFloatToUint8(rOut * 255.0))
				out.Pix[i+1] = uint8(clampFloatToUint8(gOut * 255.0))
				out.Pix[i+2] = uint8(clampFloatToUint8(bOut * 255.0))
				out.Pix[i+3] = alpha
			}
		}
		return out
	}

	// parallel path
	chunk := (h + workers - 1) / workers
	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		y0 := wi * chunk
		y1 := y0 + chunk
		if y1 > h {
			y1 = h
		}
		if y0 >= y1 {
			continue
		}
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			for y := y0; y < y1; y++ {
				for x := 0; x < w; x++ {
					i := src.PixOffset(x, y)
					alpha := src.Pix[i+3]
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

					rLin := srgb8ToLinearLUT(r)
					gLin := srgb8ToLinearLUT(g)
					bLin := srgb8ToLinearLUT(b)
					X, Y, Z := linearToXyz(rLin, gLin, bLin)
					L, aCh, bCh := xyzToLab(X, Y, Z)

					L2 := (1.0-percentage)*L + percentage*Lsep
					a2 := (1.0-percentage)*aCh + percentage*asep
					b2 := (1.0-percentage)*bCh + percentage*bsep

					x2, y2, z2 := labToXYZ(L2, a2, b2)
					r2Lin, g2Lin, b2Lin := xyzToLinearRGB(x2, y2, z2)
					rOut := linearToSrgbApprox(r2Lin)
					gOut := linearToSrgbApprox(g2Lin)
					bOut := linearToSrgbApprox(b2Lin)

					out.Pix[i+0] = uint8(clampFloatToUint8(rOut * 255.0))
					out.Pix[i+1] = uint8(clampFloatToUint8(gOut * 255.0))
					out.Pix[i+2] = uint8(clampFloatToUint8(bOut * 255.0))
					out.Pix[i+3] = alpha
				}
			}
		}(y0, y1)
	}
	wg.Wait()
	return out
}

var (
	sepiaLUTOnce    sync.Once
	srgbToLinearLUT [256]float64
	linearToSrgbLUT [256]float64 // stores sRGB(0..1) values for corresponding linear samples
)

// initSepiaLUTs initializes LUTs once.
func initSepiaLUTs() {
	sepiaLUTOnce.Do(func() {
		for i := 0; i < 256; i++ {
			v := float64(i) / 255.0
			// srgb->linear
			if v <= 0.04045 {
				srgbToLinearLUT[i] = v / 12.92
			} else {
				srgbToLinearLUT[i] = math.Pow((v+0.055)/1.055, 2.4)
			}
			// linear->srgb for a uniform linear sample value
			lv := float64(i) / 255.0
			if lv <= 0.0031308 {
				linearToSrgbLUT[i] = 12.92 * lv
			} else {
				linearToSrgbLUT[i] = 1.055*math.Pow(lv, 1.0/2.4) - 0.055
			}
		}
	})
}

func srgb8ToLinearLUT(c uint8) float64 { return srgbToLinearLUT[int(c)] }

// linearToSrgbApprox approximates linear->sRGB using LUT with interpolation
func linearToSrgbApprox(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	f := v * 255.0
	i := int(math.Floor(f))
	if i < 0 {
		i = 0
	}
	if i >= 255 {
		return linearToSrgbLUT[255]
	}
	r := linearToSrgbLUT[i]
	rn := linearToSrgbLUT[i+1]
	frac := f - float64(i)
	return r*(1-frac) + rn*frac
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
