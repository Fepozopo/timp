package stdimg

import (
	"fmt"
	"image"
	"math"
	"sort"
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
			// floor mapping to bin index
			rIdx := int(math.Floor(float64(r) * scale))
			gIdx := int(math.Floor(float64(g) * scale))
			bIdx := int(math.Floor(float64(b_) * scale))
			if rIdx >= len(rHist) {
				rIdx = len(rHist) - 1
			}
			if gIdx >= len(gHist) {
				gIdx = len(gHist) - 1
			}
			if bIdx >= len(bHist) {
				bIdx = len(bHist) - 1
			}
			rHist[rIdx]++
			gHist[gIdx]++
			bHist[bIdx]++
		}
	}
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

// RenderHistogramImage renders a smooth, filled, overlaid histogram image for the given histograms.
// It draws a dark panel, grid lines, a luminosity (gray) filled area, then colored R/G/B filled areas and strokes.
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
	if bins == 0 {
		bins = 256
	}

	// panel colors
	panelOuter := [4]uint8{20, 20, 20, 255} // very dark
	panelInner := [4]uint8{28, 28, 28, 255} // slightly lighter
	gridCol := [3]uint8{60, 60, 60}

	// create image and fill with panel outer then inner rectangle (padding)
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i+0] = panelOuter[0]
		out.Pix[i+1] = panelOuter[1]
		out.Pix[i+2] = panelOuter[2]
		out.Pix[i+3] = 255
	}
	pad := int(math.Round(float64(height) * 0.06))
	if pad < 6 {
		pad = 6
	}
	left := pad + 8
	right := width - (pad + 8)
	top := pad
	bottom := height - pad
	if left >= right || top >= bottom {
		// fallback: single padding
		left = pad
		right = width - pad
		top = pad
		bottom = height - pad
	}
	// fill inner rect
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			i := out.PixOffset(x, y)
			out.Pix[i+0] = panelInner[0]
			out.Pix[i+1] = panelInner[1]
			out.Pix[i+2] = panelInner[2]
			out.Pix[i+3] = 255
		}
	}

	plotW := right - left
	plotH := bottom - top
	if plotW <= 0 || plotH <= 0 {
		return out
	}

	// Convert histograms to float arrays resampled to plotW with linear interpolation
	resample := func(hist []int) []float64 {
		dst := make([]float64, plotW)
		if len(hist) == 0 {
			return dst
		}
		maxIdx := float64(len(hist) - 1)
		// when plotW == 1, average across all bins so the single pixel has representative value
		if plotW == 1 {
			tot := 0.0
			for _, v := range hist {
				tot += float64(v)
			}
			dst[0] = tot / float64(len(hist))
			return dst
		}
		for xi := 0; xi < plotW; xi++ {
			pos := float64(xi) * maxIdx / float64(plotW-1)
			lo := int(math.Floor(pos))
			hi := int(math.Ceil(pos))
			if lo < 0 {
				lo = 0
			}
			if hi >= len(hist) {
				hi = len(hist) - 1
			}
			if lo == hi {
				dst[xi] = float64(hist[lo])
			} else {
				frac := pos - float64(lo)
				dst[xi] = (1-frac)*float64(hist[lo]) + frac*float64(hist[hi])
			}
		}
		return dst
	}

	rF := resample(histR)
	gF := resample(histG)
	bF := resample(histB)
	// approximate luminosity histogram by weighted sum of channels per bin (use 'bins' length)
	lumBins := make([]float64, bins)
	for i := 0; i < bins; i++ {
		rv := 0.0
		gv := 0.0
		bv := 0.0
		if i < len(histR) {
			rv = float64(histR[i])
		}
		if i < len(histG) {
			gv = float64(histG[i])
		}
		if i < len(histB) {
			bv = float64(histB[i])
		}
		lumBins[i] = 0.299*rv + 0.587*gv + 0.114*bv
	}
	// resample float64 histogram to plot width
	resampleFloat := func(hist []float64) []float64 {
		dst := make([]float64, plotW)
		if len(hist) == 0 {
			return dst
		}
		maxIdx := float64(len(hist) - 1)
		if plotW == 1 {
			tot := 0.0
			for _, v := range hist {
				tot += v
			}
			dst[0] = tot / float64(len(hist))
			return dst
		}
		for xi := 0; xi < plotW; xi++ {
			pos := float64(xi) * maxIdx / float64(plotW-1)
			lo := int(math.Floor(pos))
			hi := int(math.Ceil(pos))
			if lo < 0 {
				lo = 0
			}
			if hi >= len(hist) {
				hi = len(hist) - 1
			}
			if lo == hi {
				dst[xi] = hist[lo]
			} else {
				frac := pos - float64(lo)
				dst[xi] = (1-frac)*hist[lo] + frac*hist[hi]
			}
		}
		return dst
	}
	lumF := resampleFloat(lumBins)

	// compute resampled max (what we actually draw)
	resMax := 0.0
	for _, a := range [][]float64{rF, gF, bF, lumF} {
		for _, v := range a {
			if v > resMax {
				resMax = v
			}
		}
	}
	if resMax <= 0 {
		resMax = 1.0
	}

	// Build a combined slice and use a high percentile (e.g. 99th) to avoid single outliers
	vals := make([]float64, 0, len(rF)+len(gF)+len(bF)+len(lumF))
	vals = append(vals, rF...)
	vals = append(vals, gF...)
	vals = append(vals, bF...)
	vals = append(vals, lumF...)
	sort.Float64s(vals)
	percentile := 0.99
	percentileVal := 0.0
	if len(vals) > 0 {
		idx := int(math.Floor(percentile * float64(len(vals)-1)))
		if idx < 0 {
			idx = 0
		}
		percentileVal = vals[idx]
		if percentileVal <= 0 {
			percentileVal = resMax
		}
	} else {
		percentileVal = resMax
	}

	// smoothing helper (simple moving average)
	smooth := func(a []float64, win int) []float64 {
		n := len(a)
		outS := make([]float64, n)
		if n == 0 || win <= 1 {
			copy(outS, a)
			return outS
		}
		half := win / 2
		for i := 0; i < n; i++ {
			sum := 0.0
			cnt := 0
			for j := i - half; j <= i+half; j++ {
				if j >= 0 && j < n {
					sum += a[j]
					cnt++
				}
			}
			if cnt > 0 {
				outS[i] = sum / float64(cnt)
			} else {
				outS[i] = a[i]
			}
		}
		return outS
	}

	// compute smoothed max using a 20-pixel window (helps avoid very narrow spikes)
	smoothedMax := 0.0
	for _, arr := range [][]float64{rF, gF, bF, lumF} {
		s := smooth(arr, 20)
		for _, v := range s {
			if v > smoothedMax {
				smoothedMax = v
			}
		}
	}
	if smoothedMax <= 0 {
		smoothedMax = resMax
	}

	// choose a base max that prefers smoothedMax but respects the percentile
	baseMax := math.Max(smoothedMax, percentileVal)
	if baseMax <= 0 {
		baseMax = resMax
	}
	// small headroom above the base max so curves don't butt right against the top
	capMargin := 0.05
	capMax := baseMax * (1.0 + capMargin)
	if capMax <= 0 {
		capMax = 1.0
	}

	// normalize to 0..1 using capMax
	norm := func(v float64) float64 { return v / capMax }

	// draw horizontal grid lines (4 lines) and numeric labels inside panel
	gridLines := 4

	// small 3x5 font for digits '0'..'9' (each row is 3 bits)
	font := map[rune][5]uint8{
		'0': {0x7, 0x5, 0x5, 0x5, 0x7},
		'1': {0x2, 0x6, 0x2, 0x2, 0x7},
		'2': {0x7, 0x1, 0x7, 0x4, 0x7},
		'3': {0x7, 0x1, 0x7, 0x1, 0x7},
		'4': {0x5, 0x5, 0x7, 0x1, 0x1},
		'5': {0x7, 0x4, 0x7, 0x1, 0x7},
		'6': {0x7, 0x4, 0x7, 0x5, 0x7},
		'7': {0x7, 0x1, 0x2, 0x4, 0x4},
		'8': {0x7, 0x5, 0x7, 0x5, 0x7},
		'9': {0x7, 0x5, 0x7, 0x1, 0x7},
		'.': {0x0, 0x0, 0x0, 0x6, 0x6},
	}
	drawSmallText := func(x0, y0 int, s string, col [3]uint8) {
		cx := x0
		for _, ch := range s {
			pat, ok := font[ch]
			if !ok {
				// advance by a small gap for unsupported chars
				cx += 4
				continue
			}
			for ry := 0; ry < 5; ry++ {
				row := pat[ry]
				for rx := 0; rx < 3; rx++ {
					if (row>>uint(2-rx))&1 == 0 {
						continue
					}
					px := cx + rx
					py := y0 + ry
					if px < left || px >= right || py < top || py >= bottom {
						continue
					}
					i := out.PixOffset(px, py)
					out.Pix[i+0] = col[0]
					out.Pix[i+1] = col[1]
					out.Pix[i+2] = col[2]
					out.Pix[i+3] = 255
				}
			}
			cx += 4 // char width + spacing
		}
	}

	for gi := 0; gi <= gridLines; gi++ {
		y := top + int(math.Round(float64(plotH)*float64(gi)/float64(gridLines)))
		if y < top || y >= bottom {
			continue
		}
		// line
		for x := left; x < right; x++ {
			i := out.PixOffset(x, y)
			out.Pix[i+0] = uint8(gridCol[0])
			out.Pix[i+1] = uint8(gridCol[1])
			out.Pix[i+2] = uint8(gridCol[2])
			out.Pix[i+3] = 255
		}
		// label value corresponding to this grid line (top -> capMax, bottom -> 0)
		valF := (1.0 - float64(gi)/float64(gridLines)) * capMax
		// format label with integer thousands when large
		label := fmt.Sprintf("%.0f", valF)
		labelY := y - 3
		if labelY < top {
			labelY = top
		}
		if labelY+5 >= bottom {
			labelY = bottom - 6
		}
		drawSmallText(left+4, labelY, label, [3]uint8{200, 200, 200})
	}

	// // Debug overlay: draw a thin line at the actual plotted max (resMax) and show debug numeric labels
	// plotMax := resMax
	// if plotMax > capMax {
	// 	plotMax = capMax
	// }
	// plotYP := top + (plotH - 1) - int(math.Round((plotMax/capMax)*float64(plotH-1)))
	// if plotYP < top {
	// 	plotYP = top
	// }
	// if plotYP >= bottom {
	// 	plotYP = bottom - 1
	// }
	// for x := left; x < right; x++ {
	// 	i := out.PixOffset(x, plotYP)
	// 	out.Pix[i+0] = 255
	// 	out.Pix[i+1] = 200
	// 	out.Pix[i+2] = 20
	// 	out.Pix[i+3] = 255
	// }

	// // Compose debug labels: resMax, smoothedMax, percentileVal, capMax
	// debugX := left + 6
	// debugY := top + 2
	// drawSmallText(debugX, debugY, fmt.Sprintf("%.0f", resMax), [3]uint8{255, 200, 20})
	// drawSmallText(debugX, debugY+7, fmt.Sprintf("%.0f", smoothedMax), [3]uint8{255, 140, 40})
	// drawSmallText(debugX, debugY+14, fmt.Sprintf("%.0f", percentileVal), [3]uint8{120, 200, 200})
	// drawSmallText(debugX, debugY+21, fmt.Sprintf("%.0f", capMax), [3]uint8{200, 200, 200})

	// helper blend dst, overlay with alpha (0..1)
	blendChannel := func(dst uint8, overlay uint8, alpha float64) uint8 {
		return uint8(math.Round(alpha*float64(overlay) + (1.0-alpha)*float64(dst)))
	}

	// fill area under a curve (plotX -> normalized 0..1 values) with color and alpha
	fillArea := func(values []float64, col [3]uint8, alpha float64) {
		for xi := 0; xi < plotW; xi++ {
			v := norm(values[xi])
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			yp := top + (plotH - 1) - int(math.Round(v*float64(plotH-1)))
			if yp < top {
				yp = top
			}
			if yp >= bottom {
				yp = bottom - 1
			}
			for y := yp; y < bottom; y++ {
				i := out.PixOffset(left+xi, y)
				out.Pix[i+0] = blendChannel(out.Pix[i+0], col[0], alpha)
				out.Pix[i+1] = blendChannel(out.Pix[i+1], col[1], alpha)
				out.Pix[i+2] = blendChannel(out.Pix[i+2], col[2], alpha)
			}
		}
	}

	// stroke curve (thin line) by setting a 1px colored outline with stronger alpha
	strokeCurve := func(values []float64, col [3]uint8, alpha float64) {
		for xi := 0; xi < plotW; xi++ {
			v := norm(values[xi])
			yp := top + (plotH - 1) - int(math.Round(v*float64(plotH-1)))
			if yp < top || yp >= bottom {
				continue
			}
			i := out.PixOffset(left+xi, yp)
			out.Pix[i+0] = blendChannel(out.Pix[i+0], col[0], alpha)
			out.Pix[i+1] = blendChannel(out.Pix[i+1], col[1], alpha)
			out.Pix[i+2] = blendChannel(out.Pix[i+2], col[2], alpha)
		}
	}

	// draw luminosity filled area (gray) first
	fillArea(lumF, [3]uint8{100, 110, 120}, 0.45)
	// then colored fills (lower alpha) so they tint the luminosity
	fillArea(bF, [3]uint8{65, 120, 210}, 0.20) // blue-ish
	fillArea(gF, [3]uint8{70, 200, 120}, 0.18) // green-ish
	fillArea(rF, [3]uint8{220, 90, 90}, 0.16)  // red-ish

	// strokes
	strokeCurve(bF, [3]uint8{80, 140, 230}, 0.9)
	strokeCurve(gF, [3]uint8{120, 230, 140}, 0.9)
	strokeCurve(rF, [3]uint8{250, 120, 120}, 0.9)
	// a faint stroke for luminosity
	strokeCurve(lumF, [3]uint8{180, 180, 180}, 0.35)

	return out
}
