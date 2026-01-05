package stdimg

import (
	"image"
	"math"
)

// Supported blend operators (case-insensitive): OVER, MULTIPLY, SCREEN, OVERLAY, ADD, DIFFERENCE, DISSOLVE

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func blendMultiply(sr, dr float64) float64 { return sr * dr }
func blendScreen(sr, dr float64) float64   { return 1 - (1-sr)*(1-dr) }
func blendOverlay(sr, dr float64) float64 {
	if dr < 0.5 {
		return 2 * sr * dr
	}
	return 1 - 2*(1-sr)*(1-dr)
}
func blendAdd(sr, dr float64) float64        { return clamp01(sr + dr) }
func blendDifference(sr, dr float64) float64 { return math.Abs(dr - sr) }

// Composite overlays src onto dst at offset (xoff,yoff) using operator op.
// dst is modified in place and returned.
func Composite(dst *image.NRGBA, src image.Image, op string, xoff, yoff int) *image.NRGBA {
	if dst == nil || src == nil {
		return dst
	}
	srcNR := ToNRGBA(src)
	dstB := dst.Bounds()
	srcB := srcNR.Bounds()

	// compute overlap
	startX := maxInt(dstB.Min.X, xoff)
	startY := maxInt(dstB.Min.Y, yoff)
	endX := minInt(dstB.Max.X, xoff+srcB.Dx())
	endY := minInt(dstB.Max.Y, yoff+srcB.Dy())

	if startX >= endX || startY >= endY {
		return dst // nothing to do
	}

	// choose blend function
	var blendFunc func(sr, dr float64) float64
	s := op
	s = stringUpper(s)
	switch s {
	case "MULTIPLY":
		blendFunc = blendMultiply
	case "SCREEN":
		blendFunc = blendScreen
	case "OVERLAY":
		blendFunc = blendOverlay
	case "ADD", "PLUS", "SUM":
		blendFunc = blendAdd
	case "DIFFERENCE":
		blendFunc = blendDifference
	case "DISSOLVE":
		// dissolve we'll treat as normal over; alpha of src controls dissolve
		blendFunc = func(sr, dr float64) float64 { return sr }
	default:
		// default to normal over
		blendFunc = func(sr, dr float64) float64 { return sr }
	}

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			si := srcNR.PixOffset(x-xoff, y-yoff)
			di := dst.PixOffset(x, y)
			sr := float64(srcNR.Pix[si+0]) / 255.0
			sg := float64(srcNR.Pix[si+1]) / 255.0
			sb := float64(srcNR.Pix[si+2]) / 255.0
			sa := float64(srcNR.Pix[si+3]) / 255.0

			dr_ := float64(dst.Pix[di+0]) / 255.0
			dg := float64(dst.Pix[di+1]) / 255.0
			db := float64(dst.Pix[di+2]) / 255.0
			da := float64(dst.Pix[di+3]) / 255.0

			// compute blended RGB according to blend function
			br := blendFunc(sr, dr_)
			bg := blendFunc(sg, dg)
			bb := blendFunc(sb, db)

			// composite over dst using src alpha
			outA := sa + da*(1-sa)
			outR := (1-sa)*dr_ + sa*br
			outG := (1-sa)*dg + sa*bg
			outB := (1-sa)*db + sa*bb

			// write back
			dst.Pix[di+0] = uint8(clampFloatToUint8(outR * 255.0))
			dst.Pix[di+1] = uint8(clampFloatToUint8(outG * 255.0))
			dst.Pix[di+2] = uint8(clampFloatToUint8(outB * 255.0))
			dst.Pix[di+3] = uint8(clampFloatToUint8(outA * 255.0))
		}
	}
	return dst
}

// small helpers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// stringUpper returns uppercase ASCII of s (fast path)
func stringUpper(s string) string {
	b := []byte(s)
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
