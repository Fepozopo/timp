package stdimg

import (
	"image"
	"math"
)

// MedianFilter applies a median filter with given radius (window radius).
// radius==1 -> 3x3 window. Uses a sliding-window histogram per row for O(w*h*256) worst-case but
// amortized much faster than per-pixel sorting for moderate radii.
func MedianFilter(src *image.NRGBA, radius int) *image.NRGBA {
	if src == nil {
		return nil
	}
	if radius <= 0 {
		return CloneNRGBA(src)
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(b)

	// For each row process sliding window horizontally using histograms per channel
	for y := 0; y < h; y++ {
		// precompute vertical range for this row
		y0 := y - radius
		y1 := y + radius
		if y0 < 0 {
			y0 = 0
		}
		if y1 >= h {
			y1 = h - 1
		}
		// initialize histograms for x=0 window
		rHist := [256]int{}
		gHist := [256]int{}
		bHist := [256]int{}
		aHist := [256]int{}
		windowCount := 0
		// x range for initial window
		x0 := 0 - radius
		x1 := 0 + radius
		for ox := x0; ox <= x1; ox++ {
			if ox < 0 || ox >= w {
				continue
			}
			for oy := y0; oy <= y1; oy++ {
				i := src.PixOffset(ox, oy)
				rHist[src.Pix[i+0]]++
				gHist[src.Pix[i+1]]++
				bHist[src.Pix[i+2]]++
				aHist[src.Pix[i+3]]++
				windowCount++
			}
		}
		// helper to compute initial median and cumulative for a histogram
		computeInitialMedian := func(hist *[256]int, count int) (int, int) {
			half := (count + 1) / 2
			sum := 0
			for v := 0; v < 256; v++ {
				sum += hist[v]
				if sum >= half {
					return v, sum
				}
			}
			return 0, 0
		}

		// process each x column, sliding window with running median pointers
		// maintain last medians and cumulative counts for each channel
		lastMedR, lastCumR := 0, 0
		lastMedG, lastCumG := 0, 0
		lastMedB, lastCumB := 0, 0
		lastMedA, lastCumA := 0, 0

		for x := 0; x < w; x++ {
			// For the first column (x==0) compute initial medians and cumulative sums
			if x == 0 {
				lastMedR, lastCumR = computeInitialMedian(&rHist, windowCount)
				lastMedG, lastCumG = computeInitialMedian(&gHist, windowCount)
				lastMedB, lastCumB = computeInitialMedian(&bHist, windowCount)
				lastMedA, lastCumA = computeInitialMedian(&aHist, windowCount)
			}

			// output medians
			mi := out.PixOffset(x, y)
			out.Pix[mi+0] = uint8(lastMedR)
			out.Pix[mi+1] = uint8(lastMedG)
			out.Pix[mi+2] = uint8(lastMedB)
			out.Pix[mi+3] = uint8(lastMedA)

			// slide window: remove column at x-radius, add column at x+radius+1
			removeX := x - radius
			if removeX >= 0 {
				for oy := y0; oy <= y1; oy++ {
					i := src.PixOffset(removeX, oy)
					vR := int(src.Pix[i+0])
					vG := int(src.Pix[i+1])
					vB := int(src.Pix[i+2])
					vA := int(src.Pix[i+3])
					rHist[vR]--
					gHist[vG]--
					bHist[vB]--
					aHist[vA]--
					if vR <= lastMedR {
						lastCumR--
					}
					if vG <= lastMedG {
						lastCumG--
					}
					if vB <= lastMedB {
						lastCumB--
					}
					if vA <= lastMedA {
						lastCumA--
					}
					windowCount--
				}
			}
			addX := x + radius + 1
			if addX < w {
				for oy := y0; oy <= y1; oy++ {
					i := src.PixOffset(addX, oy)
					vR := int(src.Pix[i+0])
					vG := int(src.Pix[i+1])
					vB := int(src.Pix[i+2])
					vA := int(src.Pix[i+3])
					rHist[vR]++
					gHist[vG]++
					bHist[vB]++
					aHist[vA]++
					if vR <= lastMedR {
						lastCumR++
					}
					if vG <= lastMedG {
						lastCumG++
					}
					if vB <= lastMedB {
						lastCumB++
					}
					if vA <= lastMedA {
						lastCumA++
					}
					windowCount++
				}
			}
			// adjust medians based on updated lastCum and histograms
			half := (windowCount + 1) / 2
			for lastMedR > 0 && lastCumR-rHist[lastMedR] >= half {
				lastCumR -= rHist[lastMedR]
				lastMedR--
			}
			for lastMedR < 255 && lastCumR < half {
				lastMedR++
				lastCumR += rHist[lastMedR]
			}

			for lastMedG > 0 && lastCumG-gHist[lastMedG] >= half {
				lastCumG -= gHist[lastMedG]
				lastMedG--
			}
			for lastMedG < 255 && lastCumG < half {
				lastMedG++
				lastCumG += gHist[lastMedG]
			}

			for lastMedB > 0 && lastCumB-bHist[lastMedB] >= half {
				lastCumB -= bHist[lastMedB]
				lastMedB--
			}
			for lastMedB < 255 && lastCumB < half {
				lastMedB++
				lastCumB += bHist[lastMedB]
			}

			for lastMedA > 0 && lastCumA-aHist[lastMedA] >= half {
				lastCumA -= aHist[lastMedA]
				lastMedA--
			}
			for lastMedA < 255 && lastCumA < half {
				lastMedA++
				lastCumA += aHist[lastMedA]
			}
		}
	}
	return out
}

// Level applies levels adjustment. blackPoint and whitePoint are in [0..255] range,
// gamma is a multiplier (ImageMagick uses gamma as midtone adjustment). We'll implement
// mapping: normalized = (v - black)/(white - black) clamped to [0,1]; if gamma != 0, apply pow(normalized, 1.0/gamma).
func Level(src *image.NRGBA, blackPoint, gamma, whitePoint float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	out := CloneNRGBA(src)
	minV := blackPoint
	maxV := whitePoint
	if maxV <= minV {
		// no-op
		return out
	}
	invGamma := 1.0
	if gamma > 0 {
		invGamma = 1.0 / gamma
	}
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			a := float64(src.Pix[i+3])

			rn := math.Min(math.Max((r-minV)/(maxV-minV), 0.0), 1.0)
			gn := math.Min(math.Max((g-minV)/(maxV-minV), 0.0), 1.0)
			bn := math.Min(math.Max((b_-minV)/(maxV-minV), 0.0), 1.0)

			if gamma > 0 {
				rn = math.Pow(rn, invGamma)
				gn = math.Pow(gn, invGamma)
				bn = math.Pow(bn, invGamma)
			}

			out.Pix[i+0] = uint8(clampFloatToUint8(rn * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(gn * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(bn * 255.0))
			out.Pix[i+3] = uint8(clampFloatToUint8(a))
		}
	}
	return out
}

// Gamma applies per-channel gamma correction (gamma>0). gamma==1 -> no-op
func Gamma(src *image.NRGBA, gamma float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	if gamma <= 0 || math.IsNaN(gamma) || math.IsInf(gamma, 0) {
		return CloneNRGBA(src)
	}
	inv := 1.0 / gamma
	b := src.Bounds()
	out := CloneNRGBA(src)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0]) / 255.0
			g := float64(src.Pix[i+1]) / 255.0
			b_ := float64(src.Pix[i+2]) / 255.0
			a := src.Pix[i+3]
			out.Pix[i+0] = uint8(clampFloatToUint8(math.Pow(r, inv) * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(math.Pow(g, inv) * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(math.Pow(b_, inv) * 255.0))
			out.Pix[i+3] = a
		}
	}
	return out
}

// Negate inverts colors; if onlyGray true, invert only luminance and keep color channels mapped accordingly.
func Negate(src *image.NRGBA, onlyGray bool) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	out := CloneNRGBA(src)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := src.Pix[i+0]
			g := src.Pix[i+1]
			b_ := src.Pix[i+2]
			a := src.Pix[i+3]
			if onlyGray {
				// compute luminance, invert to 255-lum, then scale channels proportionally
				rf := float64(r) / 255.0
				gf := float64(g) / 255.0
				bf := float64(b_) / 255.0
				lum := 0.2126*rf + 0.7152*gf + 0.0722*bf
				invLum := 1.0 - lum
				if lum <= 0 {
					out.Pix[i+0] = uint8(clampFloatToUint8(invLum * 255.0))
					out.Pix[i+1] = out.Pix[i+0]
					out.Pix[i+2] = out.Pix[i+0]
				} else {
					rScale := (rf / lum)
					gScale := (gf / lum)
					bScale := (bf / lum)
					out.Pix[i+0] = uint8(clampFloatToUint8(invLum * rScale * 255.0))
					out.Pix[i+1] = uint8(clampFloatToUint8(invLum * gScale * 255.0))
					out.Pix[i+2] = uint8(clampFloatToUint8(invLum * bScale * 255.0))
				}
				out.Pix[i+3] = a
			} else {
				out.Pix[i+0] = 255 - r
				out.Pix[i+1] = 255 - g
				out.Pix[i+2] = 255 - b_
				out.Pix[i+3] = a
			}
		}
	}
	return out
}

// Threshold applies a binary threshold on luminance (if perChannel false) or per-channel (if true).
func Threshold(src *image.NRGBA, thresh float64, perChannel bool) *image.NRGBA {
	if src == nil {
		return nil
	}
	if thresh < 0 {
		thresh = 0
	}
	if thresh > 255 {
		thresh = 255
	}
	b := src.Bounds()
	out := CloneNRGBA(src)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := src.Pix[i+0]
			g := src.Pix[i+1]
			b_ := src.Pix[i+2]
			a := src.Pix[i+3]
			if perChannel {
				if float64(r) >= thresh {
					out.Pix[i+0] = 255
				} else {
					out.Pix[i+0] = 0
				}
				if float64(g) >= thresh {
					out.Pix[i+1] = 255
				} else {
					out.Pix[i+1] = 0
				}
				if float64(b_) >= thresh {
					out.Pix[i+2] = 255
				} else {
					out.Pix[i+2] = 0
				}
				out.Pix[i+3] = a
			} else {
				// luminance threshold
				rf := float64(r) / 255.0
				gf := float64(g) / 255.0
				bf := float64(b_) / 255.0
				lum := 0.2126*rf + 0.7152*gf + 0.0722*bf
				if lum*255.0 >= thresh {
					out.Pix[i+0] = 255
					out.Pix[i+1] = 255
					out.Pix[i+2] = 255
				} else {
					out.Pix[i+0] = 0
					out.Pix[i+1] = 0
					out.Pix[i+2] = 0
				}
				out.Pix[i+3] = a
			}
		}
	}
	return out
}

// Normalize stretches per-channel extremes to full [0,255] range.
func Normalize(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	minR, minG, minB := 255.0, 255.0, 255.0
	maxR, maxG, maxB := 0.0, 0.0, 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			if r < minR {
				minR = r
			}
			if g < minG {
				minG = g
			}
			if b_ < minB {
				minB = b_
			}
			if r > maxR {
				maxR = r
			}
			if g > maxG {
				maxG = g
			}
			if b_ > maxB {
				maxB = b_
			}
		}
	}
	out := image.NewNRGBA(b)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b_ := float64(src.Pix[i+2])
			a := src.Pix[i+3]

			var rn, gn, bn float64
			if maxR <= minR {
				rn = r / 255.0
			} else {
				rn = (r - minR) / (maxR - minR)
			}
			if maxG <= minG {
				gn = g / 255.0
			} else {
				gn = (g - minG) / (maxG - minG)
			}
			if maxB <= minB {
				bn = b_ / 255.0
			} else {
				bn = (b_ - minB) / (maxB - minB)
			}

			out.Pix[i+0] = uint8(clampFloatToUint8(rn * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(gn * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(bn * 255.0))
			out.Pix[i+3] = a
		}
	}
	return out
}

// AutoLevel: convenience wrapper for Normalize
func AutoLevel(src *image.NRGBA) *image.NRGBA {
	return Normalize(src)
}

// AutoGamma estimates a gamma to map the average luminance to 0.5 and applies it.
func AutoGamma(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	total := float64(w * h)
	if total <= 0 {
		return CloneNRGBA(src)
	}
	// compute mean luminance in [0,1]
	mean := 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0]) / 255.0
			g := float64(src.Pix[i+1]) / 255.0
			b_ := float64(src.Pix[i+2]) / 255.0
			// Rec.709 luminance
			lum := 0.2126*r + 0.7152*g + 0.0722*b_
			mean += lum
		}
	}
	mean = mean / total
	if mean <= 0 || mean >= 1 {
		return CloneNRGBA(src)
	}
	// solve for gamma such that mean^gamma = 0.5 => gamma = log(0.5)/log(mean)
	gamma := math.Log(0.5) / math.Log(mean)
	// clamp gamma to reasonable range
	if math.IsNaN(gamma) || math.IsInf(gamma, 0) {
		return CloneNRGBA(src)
	}
	if gamma < 0.1 {
		gamma = 0.1
	}
	if gamma > 10 {
		gamma = 10
	}
	// apply gamma: out = (v/255)^{gamma} * 255
	out := image.NewNRGBA(b)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0]) / 255.0
			g := float64(src.Pix[i+1]) / 255.0
			b_ := float64(src.Pix[i+2]) / 255.0
			a := src.Pix[i+3]
			out.Pix[i+0] = uint8(clampFloatToUint8(math.Pow(r, gamma) * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(math.Pow(g, gamma) * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(math.Pow(b_, gamma) * 255.0))
			out.Pix[i+3] = a
		}
	}
	return out
}
