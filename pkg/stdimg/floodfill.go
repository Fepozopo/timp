package stdimg

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

// color conversion helpers: sRGB -> linear -> XYZ -> Lab
func srgbToLinear(c uint8) float64 {
	v := float64(c) / 255.0
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToXyz(r, g, b float64) (x, y, z float64) {
	// sRGB D65 matrix
	x = 0.4124564*r + 0.3575761*g + 0.1804375*b
	y = 0.2126729*r + 0.7151522*g + 0.0721750*b
	z = 0.0193339*r + 0.1191920*g + 0.9503041*b
	return
}

func xyzToLab(x, y, z float64) (l, a, b float64) {
	// reference D65
	xr := x / 0.95047
	yr := y / 1.00000
	zr := z / 1.08883
	f := func(t float64) float64 {
		if t > 0.008856 {
			return math.Pow(t, 1.0/3.0)
		}
		return 7.787037*t + 16.0/116.0
	}
	fx := f(xr)
	fy := f(yr)
	fz := f(zr)
	l = 116.0*fy - 16.0
	a = 500.0 * (fx - fy)
	b = 200.0 * (fy - fz)
	return
}

func rgbToLab(c color.NRGBA) (l, a, b float64) {
	r := srgbToLinear(c.R)
	g := srgbToLinear(c.G)
	bl := srgbToLinear(c.B)
	x, y, z := linearToXyz(r, g, bl)
	return xyzToLab(x, y, z)
}

func labDistanceSq(c1, c2 color.NRGBA) float64 {
	l1, a1, b1 := rgbToLab(c1)
	l2, a2, b2 := rgbToLab(c2)
	dl := l1 - l2
	da := a1 - a2
	db := b1 - b2
	return dl*dl + da*da + db*db
}

// FloodfillPaint fills a region starting at (x,y) with fillColor using a fuzz tolerance.
// Optimized for large images by using a bitset for the mask (1 bit per pixel).
// If borderColor has non-zero alpha or non-zero RGB, it is treated as a boundary that cannot be crossed (within fuzz).
// If invert is true, fill the inverse of the matched region.
func FloodfillPaint(src *image.NRGBA, fillColor color.NRGBA, fuzz float64, borderColor color.NRGBA, x, y int, invert bool) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	// clamp start
	if x < b.Min.X {
		x = b.Min.X
	}
	if x >= b.Max.X {
		x = b.Max.X - 1
	}
	if y < b.Min.Y {
		y = b.Min.Y
	}
	if y >= b.Max.Y {
		y = b.Max.Y - 1
	}

	// prepare mask as bitset to reduce memory (1 bit per pixel)
	size := w * h
	maskBytes := (size + 7) / 8
	mask := make([]byte, maskBytes)

	// bit helpers
	getMask := func(i int) byte {
		return (mask[i>>3] >> (uint(i) & 7)) & 1
	}
	setMask := func(i int) {
		mask[i>>3] |= 1 << (uint(i) & 7)
	}
	clearMask := func(i int) {
		mask[i>>3] &^= 1 << (uint(i) & 7)
	}

	// helpers
	idxOf := func(px, py int) int { return (py-b.Min.Y)*w + (px - b.Min.X) }

	// starting pixel and border decision
	start := samplePixelClamped(src, x, y)
	useBorder := !(borderColor.R == 0 && borderColor.G == 0 && borderColor.B == 0 && borderColor.A == 0)

	// clamp fuzz (interpreted as Lab Delta-E units)
	if fuzz < 0 {
		fuzz = 0
	}
	if fuzz > 200 {
		fuzz = 200
	}
	fuzzSq := fuzz * fuzz

	// boundary test using perceptual Lab distance
	var isBoundary func(c color.NRGBA) bool
	if useBorder {
		isBoundary = func(c color.NRGBA) bool { return labDistanceSq(c, borderColor) <= fuzzSq }
	} else {
		isBoundary = func(c color.NRGBA) bool { return false }
	}

	// matching test using Lab distance (unless using border mode)
	matchesTarget := func(c color.NRGBA) bool {
		if useBorder {
			// non-boundary pixels are fillable in border mode
			return !isBoundary(c)
		}
		return labDistanceSq(c, start) <= fuzzSq
	}

	// If invert without border, treat matching as global (non-connected): mark all matching pixels
	if !useBorder && invert {
		for py := b.Min.Y; py < b.Max.Y; py++ {
			for px := b.Min.X; px < b.Max.X; px++ {
				i := idxOf(px, py)
				c := samplePixelClamped(src, px, py)
				if matchesTarget(c) {
					setMask(i)
				}
			}
		}
	} else {
		// Scanline / span flood-fill (8-way connectivity): stack of seed points
		type seed struct{ x, y int }
		stackSeeds := make([]seed, 0, 1024)
		stackSeeds = append(stackSeeds, seed{x: x, y: y})
		minX := b.Min.X
		maxX := b.Max.X
		minY := b.Min.Y
		maxY := b.Max.Y
		for len(stackSeeds) > 0 {
			// pop
			s := stackSeeds[len(stackSeeds)-1]
			stackSeeds = stackSeeds[:len(stackSeeds)-1]
			sx := s.x
			sy := s.y
			if sx < minX || sx >= maxX || sy < minY || sy >= maxY {
				continue
			}
			i0 := idxOf(sx, sy)
			if getMask(i0) == 1 {
				continue
			}
			c0 := samplePixelClamped(src, sx, sy)
			if isBoundary(c0) || !matchesTarget(c0) {
				continue
			}
			// expand left
			xl := sx
			for xl-1 >= minX {
				i := idxOf(xl-1, sy)
				if getMask(i) == 1 {
					break
				}
				c := samplePixelClamped(src, xl-1, sy)
				if isBoundary(c) || !matchesTarget(c) {
					break
				}
				xl--
			}
			// expand right
			xr := sx
			for xr+1 < maxX {
				i := idxOf(xr+1, sy)
				if getMask(i) == 1 {
					break
				}
				c := samplePixelClamped(src, xr+1, sy)
				if isBoundary(c) || !matchesTarget(c) {
					break
				}
				xr++
			}
			// set mask for span xl..xr
			for xi := xl; xi <= xr; xi++ {
				setMask(idxOf(xi, sy))
			}
			// push seeds for adjacent rows: y-1 and y+1
			for adjY := sy - 1; adjY <= sy+1; adjY += 2 {
				if adjY < minY || adjY >= maxY {
					continue
				}
				// for 8-way connectivity consider xl-1..xr+1 on adjacent row
				startX := xl - 1
				if startX < minX {
					startX = minX
				}
				endX := xr + 1
				if endX >= maxX {
					endX = maxX - 1
				}
				x := startX
				for x <= endX {
					// skip already-set segments
					idx := idxOf(x, adjY)
					if getMask(idx) == 1 {
						x++
						continue
					}
					c := samplePixelClamped(src, x, adjY)
					if isBoundary(c) || !matchesTarget(c) {
						x++
						continue
					}
					// found a new seed at (x,adjY)
					stackSeeds = append(stackSeeds, seed{x: x, y: adjY})
					// skip over contiguous run on this adjacent row
					for x++; x <= endX; x++ {
						idx2 := idxOf(x, adjY)
						if getMask(idx2) == 1 {
							break
						}
						c2 := samplePixelClamped(src, x, adjY)
						if isBoundary(c2) || !matchesTarget(c2) {
							break
						}
					}
				}
			}
		}
	}

	// invert mask if requested (flip bits)
	if invert {
		for i := 0; i < size; i++ {
			if getMask(i) == 1 {
				clearMask(i)
			} else {
				setMask(i)
			}
		}
	}

	// composite fillColor over src where mask==1
	out := CloneNRGBA(src)
	fr := float64(fillColor.R)
	fg := float64(fillColor.G)
	fb := float64(fillColor.B)
	fa := float64(fillColor.A) / 255.0

	// Parallelize compositing by splitting rows across workers
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	rowsPer := (h + workers - 1) / workers
	for wi := 0; wi < workers; wi++ {
		startRow := b.Min.Y + wi*rowsPer
		endRow := startRow + rowsPer
		if endRow > b.Min.Y+h {
			endRow = b.Min.Y + h
		}
		go func(startY, endY int) {
			defer wg.Done()
			for py := startY; py < endY; py++ {
				for px := b.Min.X; px < b.Max.X; px++ {
					i := idxOf(px, py)
					if getMask(i) == 0 {
						continue
					}
					off := out.PixOffset(px, py)
					sr := float64(src.Pix[off+0])
					sg := float64(src.Pix[off+1])
					sb := float64(src.Pix[off+2])
					sa := float64(src.Pix[off+3]) / 255.0

					outA := fa + sa*(1.0-fa)
					var nr, ng, nb float64
					if outA > 0 {
						nr = (fr*fa + sr*sa*(1.0-fa)) / outA
						ng = (fg*fa + sg*sa*(1.0-fa)) / outA
						nb = (fb*fa + sb*sa*(1.0-fa)) / outA
					} else {
						nr, ng, nb = 0, 0, 0
					}
					out.Pix[off+0] = uint8(clampFloatToUint8(nr))
					out.Pix[off+1] = uint8(clampFloatToUint8(ng))
					out.Pix[off+2] = uint8(clampFloatToUint8(nb))
					out.Pix[off+3] = uint8(clampFloatToUint8(outA * 255.0))
				}
			}
		}(startRow, endRow)
	}
	wg.Wait()

	return out
}
