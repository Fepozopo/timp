package stdimg

import (
	"image"
	"image/color"
)

// ToNRGBA converts any image.Image to *image.NRGBA (non-premultiplied RGBA).
func ToNRGBA(src image.Image) *image.NRGBA {
	if src == nil {
		return nil
	}
	if n, ok := src.(*image.NRGBA); ok {
		// return a copy to avoid modifying original
		out := image.NewNRGBA(n.Rect)
		copy(out.Pix, n.Pix)
		return out
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	idx := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b_, a := src.At(x, y).RGBA()
			// r,g,b,a are 16-bit [0, 65535]; convert to 8-bit
			out.Pix[idx+0] = uint8(r >> 8)
			out.Pix[idx+1] = uint8(g >> 8)
			out.Pix[idx+2] = uint8(b_ >> 8)
			out.Pix[idx+3] = uint8(a >> 8)
			idx += 4
		}
	}
	return out
}

// CloneNRGBA returns a copy of the provided image.NRGBA
func CloneNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	out := image.NewNRGBA(src.Rect)
	copy(out.Pix, src.Pix)
	return out
}

// clampInt clamps v to [lo,hi]
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// samplePixelClamped returns the color.NRGBA at integer coords clamped to image.
func samplePixelClamped(img *image.NRGBA, x, y int) color.NRGBA {
	b := img.Bounds()
	x = clampInt(x, b.Min.X, b.Max.X-1)
	y = clampInt(y, b.Min.Y, b.Max.Y-1)
	i := img.PixOffset(x, y)
	return color.NRGBA{img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
}

func makeSolidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i+0] = c.R
			img.Pix[i+1] = c.G
			img.Pix[i+2] = c.B
			img.Pix[i+3] = c.A
		}
	}
	return img
}
