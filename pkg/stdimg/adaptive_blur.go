package stdimg

import (
	"image"
	"math"
	"runtime"
)

// AdaptiveBlur approximates an adaptive blur by blending a blurred image with the original
// using an edge-based mask. This function remains as a simple approximation and calls
// AdaptiveBlurPerPixel for a true variance-driven implementation when appropriate.
func AdaptiveBlur(src *image.NRGBA, radius, sigma float64) *image.NRGBA {
	// For backward compatibility, use the true per-pixel implementation with reasonable defaults
	return AdaptiveBlurPerPixel(src, radius, 0.5, sigma, 6)
}

// AdaptiveBlurPerPixel implements a variance-driven per-pixel adaptive blur.
// radius controls the neighborhood used to compute local variance (in pixels).
// sigmaMin/sigmaMax define the range of Gaussian sigmas to apply (sigmaMin for high-variance regions, sigmaMax for low-variance areas).
// levels is the number of discrete sigma levels to precompute and is used to approximate per-pixel variable sigma efficiently.
func AdaptiveBlurPerPixel(src *image.NRGBA, radius, sigmaMin, sigmaMax float64, levels int) *image.NRGBA {
	if src == nil {
		return nil
	}
	if levels <= 1 || sigmaMin == sigmaMax {
		// fallback to uniform blur with sigmaMax
		return SeparableGaussianBlur(src, sigmaMax)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	// compute luminance image
	lum := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0]) / 255.0
			g := float64(src.Pix[i+1]) / 255.0
			b_ := float64(src.Pix[i+2]) / 255.0
			lum[y*w+x] = 0.2126*r + 0.7152*g + 0.0722*b_
		}
	}
	// build integral images for sum and sumsq (size (w+1)*(h+1))
	sz := (w + 1) * (h + 1)
	sum := make([]float64, sz)
	sumsq := make([]float64, sz)
	for y := 0; y < h; y++ {
		rowSum := 0.0
		rowSumSq := 0.0
		for x := 0; x < w; x++ {
			v := lum[y*w+x]
			rowSum += v
			rowSumSq += v * v
			idx := (y+1)*(w+1) + (x + 1)
			sum[idx] = sum[idx-(w+1)] + rowSum
			sumsq[idx] = sumsq[idx-(w+1)] + rowSumSq
		}
	}
	// helper to get sum over rect x0..x1, y0..y1 inclusive
	rectSum := func(x0, y0, x1, y1 int) float64 {
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 >= w {
			x1 = w - 1
		}
		if y1 >= h {
			y1 = h - 1
		}
		A := (y0)*(w+1) + x0
		B := (y0)*(w+1) + (x1 + 1)
		C := (y1+1)*(w+1) + x0
		D := (y1+1)*(w+1) + (x1 + 1)
		return sum[D] - sum[B] - sum[C] + sum[A]
	}
	rectSumSq := func(x0, y0, x1, y1 int) float64 {
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 >= w {
			x1 = w - 1
		}
		if y1 >= h {
			y1 = h - 1
		}
		A := (y0)*(w+1) + x0
		B := (y0)*(w+1) + (x1 + 1)
		C := (y1+1)*(w+1) + x0
		D := (y1+1)*(w+1) + (x1 + 1)
		return sumsq[D] - sumsq[B] - sumsq[C] + sumsq[A]
	}
	// compute local variances and track min/max
	vars := make([]float64, w*h)
	minVar := math.Inf(1)
	maxVar := 0.0
	r := int(math.Max(1.0, math.Floor(radius)))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			x0 := x - r
			y0 := y - r
			x1 := x + r
			y1 := y + r
			area := float64((x1 - x0 + 1) * (y1 - y0 + 1))
			s := rectSum(x0, y0, x1, y1)
			s2 := rectSumSq(x0, y0, x1, y1)
			mean := s / area
			variance := s2/area - mean*mean
			if variance < 0 {
				variance = 0
			}
			vars[y*w+x] = variance
			if variance < minVar {
				minVar = variance
			}
			if variance > maxVar {
				maxVar = variance
			}
		}
	}
	// prepare sigma levels
	levelSig := make([]float64, levels)
	for i := 0; i < levels; i++ {
		levelSig[i] = sigmaMin + (sigmaMax-sigmaMin)*float64(i)/float64(levels-1)
	}
	// precompute blurred images for each sigma level
	blurredLevels := make([]*image.NRGBA, levels)
	for i := 0; i < levels; i++ {
		blurredLevels[i] = SeparableGaussianBlur(src, levelSig[i])
	}
	// build output by interpolating between adjacent levels per pixel based on normalized variance (parallelized)
	out := image.NewNRGBA(b)
	// choose worker count based on available CPUs
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > h {
		workers = h
	}
	rowsPerWorker := (h + workers - 1) / workers
	done := make(chan struct{}, workers)
	for worker := 0; worker < workers; worker++ {
		y0 := worker * rowsPerWorker
		y1 := y0 + rowsPerWorker
		if y1 > h {
			y1 = h
		}
		if y0 >= y1 {
			// nothing for this worker
			done <- struct{}{}
			continue
		}
		go func(y0, y1 int) {
			for y := y0; y < y1; y++ {
				for x := 0; x < w; x++ {
					v := vars[y*w+x]
					varNorm := 0.0
					if maxVar > minVar {
						varNorm = (v - minVar) / (maxVar - minVar)
						if varNorm < 0 {
							varNorm = 0
						}
						if varNorm > 1 {
							varNorm = 1
						}
					}
					// fractional index: higher variance -> closer to sigmaMin (less blur), so invert varNorm
					idxF := (1.0 - varNorm) * float64(levels-1)
					idx0 := int(math.Floor(idxF))
					if idx0 < 0 {
						idx0 = 0
					}
					if idx0 >= levels-1 {
						// use last level directly
						pixOff := out.PixOffset(x, y)
						bPixOff := blurredLevels[levels-1].PixOffset(x, y)
						out.Pix[pixOff+0] = blurredLevels[levels-1].Pix[bPixOff+0]
						out.Pix[pixOff+1] = blurredLevels[levels-1].Pix[bPixOff+1]
						out.Pix[pixOff+2] = blurredLevels[levels-1].Pix[bPixOff+2]
						out.Pix[pixOff+3] = src.Pix[src.PixOffset(x, y)+3]
						continue
					}
					idx1 := idx0 + 1
					t := idxF - float64(idx0)
					// interpolate between blurredLevels[idx0] and blurredLevels[idx1]
					pixOff := out.PixOffset(x, y)
					b0 := blurredLevels[idx0].PixOffset(x, y)
					b1 := blurredLevels[idx1].PixOffset(x, y)
					r0 := float64(blurredLevels[idx0].Pix[b0+0])
					g0 := float64(blurredLevels[idx0].Pix[b0+1])
					b_0 := float64(blurredLevels[idx0].Pix[b0+2])
					r1 := float64(blurredLevels[idx1].Pix[b1+0])
					g1 := float64(blurredLevels[idx1].Pix[b1+1])
					b_1 := float64(blurredLevels[idx1].Pix[b1+2])
					rVal := r0*(1.0-t) + r1*t
					gVal := g0*(1.0-t) + g1*t
					bVal := b_0*(1.0-t) + b_1*t
					out.Pix[pixOff+0] = uint8(clampFloatToUint8(rVal))
					out.Pix[pixOff+1] = uint8(clampFloatToUint8(gVal))
					out.Pix[pixOff+2] = uint8(clampFloatToUint8(bVal))
					out.Pix[pixOff+3] = src.Pix[src.PixOffset(x, y)+3]
				}
			}
			done <- struct{}{}
		}(y0, y1)
	}
	// wait for workers
	for i := 0; i < workers; i++ {
		<-done
	}
	close(done)
	return out
}
