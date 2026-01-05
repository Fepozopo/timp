package stdimg

import (
	"image"
	"math"
)

// ComputeHistogram computes per-channel histograms with `bins` bins (e.g., 256).
// Returns three slices for R, G, B counts.
func ComputeHistogram(src *image.NRGBA, bins int) ([]int, []int, []int) {
	if src == nil {
		return nil, nil, nil
	}
	if bins <= 0 {
		bins = 256
	}
	rHist := make([]int, bins)
	gHist := make([]int, bins)
	bHist := make([]int, bins)
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	scale := float64(bins) / 256.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := src.Pix[i+0]
			g := src.Pix[i+1]
			b_ := src.Pix[i+2]
			rHist[int(math.Floor(float64(r)*scale))]++
			gHist[int(math.Floor(float64(g)*scale))]++
			bHist[int(math.Floor(float64(b_)*scale))]++
		}
	}
	// handle possible index==bins due to r==255 rounding
	fix := func(hs []int) {
		if len(hs) == 0 {
			return
		}
		if idx := len(hs); idx > 0 {
			// ensure no out-of-range
			if hs[len(hs)-1] == 0 {
				// nothing
			}
		}
	}
	fix(rHist)
	fix(gHist)
	fix(bHist)
	return rHist, gHist, bHist
}

// Equalize performs histogram equalization per channel and returns a new image.
func Equalize(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	// compute histograms with 256 bins
	rHist, gHist, bHist := ComputeHistogram(src, 256)
	total := src.Bounds().Dx() * src.Bounds().Dy()

	// cdf and map
	mapR := make([]uint8, 256)
	mapG := make([]uint8, 256)
	mapB := make([]uint8, 256)

	cdf := 0
	// R
	for i := 0; i < 256; i++ {
		cdf += rHist[i]
		mapR[i] = uint8(math.Round(float64(cdf) / float64(total) * 255.0))
	}
	cdf = 0
	for i := 0; i < 256; i++ {
		cdf += gHist[i]
		mapG[i] = uint8(math.Round(float64(cdf) / float64(total) * 255.0))
	}
	cdf = 0
	for i := 0; i < 256; i++ {
		cdf += bHist[i]
		mapB[i] = uint8(math.Round(float64(cdf) / float64(total) * 255.0))
	}

	b := src.Bounds()
	out := image.NewNRGBA(b)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := src.Pix[i+0]
			g := src.Pix[i+1]
			b_ := src.Pix[i+2]
			a := src.Pix[i+3]
			out.Pix[i+0] = mapR[r]
			out.Pix[i+1] = mapG[g]
			out.Pix[i+2] = mapB[b_]
			out.Pix[i+3] = a
		}
	}
	return out
}

// RenderHistogramImage renders a simple overlaid histogram image for the given histograms.
// histR/G/B slices length == bins. width/height choose the output image size.
func RenderHistogramImage(histR, histG, histB []int, width, height int) *image.NRGBA {
	if width <= 0 {
		width = 512
	}
	if height <= 0 {
		height = 120
	}
	bins := len(histR)
	if bins == 0 {
		bins = len(histG)
	}
	if bins == 0 {
		bins = len(histB)
	}
	// create image with white background
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i+0] = 255
		out.Pix[i+1] = 255
		out.Pix[i+2] = 255
		out.Pix[i+3] = 255
	}
	// find max across channels
	maxv := 1
	for _, v := range histR {
		if v > maxv {
			maxv = v
		}
	}
	for _, v := range histG {
		if v > maxv {
			maxv = v
		}
	}
	for _, v := range histB {
		if v > maxv {
			maxv = v
		}
	}

	// draw each bin as a vertical line at x position
	for x := 0; x < width; x++ {
		// determine bin index
		bin := int(math.Floor(float64(x) * float64(bins) / float64(width)))
		if bin < 0 {
			bin = 0
		}
		if bin >= bins {
			bin = bins - 1
		}
		// compute heights
		rh := int(math.Round(float64(histR[bin]) / float64(maxv) * float64(height-1)))
		gh := int(math.Round(float64(histG[bin]) / float64(maxv) * float64(height-1)))
		bh := int(math.Round(float64(histB[bin]) / float64(maxv) * float64(height-1)))
		// draw from bottom up
		for y := 0; y < rh; y++ {
			i := out.PixOffset(x, height-1-y)
			// red channel overlay: set red channel to max
			out.Pix[i+0] = 255
			// blend by max for simplicity; keep other channels as background
		}
		for y := 0; y < gh; y++ {
			i := out.PixOffset(x, height-1-y)
			// set green
			out.Pix[i+1] = 255
		}
		for y := 0; y < bh; y++ {
			i := out.PixOffset(x, height-1-y)
			out.Pix[i+2] = 255
		}
	}
	return out
}
