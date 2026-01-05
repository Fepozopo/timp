package stdimg

import (
	"image"
	"image/draw"
)

// Trim removes uniform border regions matching the top-left pixel color within a fuzz tolerance.
// fuzz is an absolute color distance on 0..255 scale (Euclidean distance across RGB).
func Trim(src *image.NRGBA, fuzz float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w == 0 || h == 0 {
		return CloneNRGBA(src)
	}
	// reference color: top-left corner
	refX := b.Min.X
	refY := b.Min.Y
	refo := src.PixOffset(refX, refY)
	refR := float64(src.Pix[refo+0])
	refG := float64(src.Pix[refo+1])
	refB := float64(src.Pix[refo+2])
	// refA := float64(src.Pix[refo+3])
	fuzzSq := fuzz * fuzz

	minX := b.Max.X
	minY := b.Max.Y
	maxX := b.Min.X - 1
	maxY := b.Min.Y - 1

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			dx := r - refR
			dy := g - refG
			dz := b_ - refB
			dsq := dx*dx + dy*dy + dz*dz
			if dsq > fuzzSq {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// If nothing differs, return original
	if maxX < minX || maxY < minY {
		return CloneNRGBA(src)
	}

	rect := image.Rect(minX, minY, maxX+1, maxY+1)
	out := image.NewNRGBA(rect)
	draw.Draw(out, rect.Sub(rect.Min), src, rect.Min, draw.Src)
	return out
}
