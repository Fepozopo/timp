package stdimg

import (
	"image"
)

// AdaptiveThreshold applies a local mean threshold over a window of windowW x windowH.
// Pixels above mean - offset become white, otherwise black. Returns a bilevel NRGBA image.
func AdaptiveThreshold(src *image.NRGBA, windowW, windowH int, offset float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if windowW <= 0 {
		windowW = 15
	}
	if windowH <= 0 {
		windowH = 15
	}
	// compute luminance image as float64
	lum := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x+b.Min.X, y+b.Min.Y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			lum[y*w+x] = 0.2126*r + 0.7152*g + 0.0722*b_
		}
	}
	// integral image for fast local means
	integ := make([]float64, (w+1)*(h+1))
	for y := 1; y <= h; y++ {
		sum := 0.0
		for x := 1; x <= w; x++ {
			sum += lum[(y-1)*w+(x-1)]
			integ[y*(w+1)+x] = integ[(y-1)*(w+1)+x] + sum
		}
	}

	out := image.NewNRGBA(src.Rect)
	halfW := windowW / 2
	halfH := windowH / 2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			x0 := clampInt(x-halfW, 0, w-1)
			x1 := clampInt(x+halfW, 0, w-1)
			y0 := clampInt(y-halfH, 0, h-1)
			y1 := clampInt(y+halfH, 0, h-1)
			// integral coordinates are +1
			sx := x0 + 1
			ex := x1 + 1
			sy := y0 + 1
			ey := y1 + 1
			area := float64((x1 - x0 + 1) * (y1 - y0 + 1))
			s := integ[ey*(w+1)+ex] - integ[(sy-1)*(w+1)+ex] - integ[ey*(w+1)+(sx-1)] + integ[(sy-1)*(w+1)+(sx-1)]
			mean := s / area
			th := mean - offset
			idx := src.PixOffset(x+b.Min.X, y+b.Min.Y)
			val := 0.2126*float64(src.Pix[idx+0]) + 0.7152*float64(src.Pix[idx+1]) + 0.0722*float64(src.Pix[idx+2])
			if val > th {
				out.Pix[idx+0] = 255
				out.Pix[idx+1] = 255
				out.Pix[idx+2] = 255
				out.Pix[idx+3] = src.Pix[idx+3]
			} else {
				out.Pix[idx+0] = 0
				out.Pix[idx+1] = 0
				out.Pix[idx+2] = 0
				out.Pix[idx+3] = src.Pix[idx+3]
			}
		}
	}
	return out
}
