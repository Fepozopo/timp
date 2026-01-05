package stdimg

import (
	"image"
	"testing"
)

func TestAdaptiveThresholdBasic(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	// create a horizontal gradient
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			i := src.PixOffset(x, y)
			v := uint8(x * 32)
			src.Pix[i+0] = v
			src.Pix[i+1] = v
			src.Pix[i+2] = v
			src.Pix[i+3] = 255
		}
	}
	out := AdaptiveThreshold(src, 3, 3, 0.0)
	if out == nil {
		t.Fatal("output nil")
	}
	// verify only two colors present (0 or 255 in R channel)
	seen0 := false
	seen255 := false
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			idx := out.PixOffset(x, y)
			r := out.Pix[idx+0]
			if r == 0 {
				seen0 = true
			} else if r == 255 {
				seen255 = true
			} else {
				t.Fatalf("unexpected value: %d", r)
			}
		}
	}
	if !seen0 || !seen255 {
		t.Fatalf("expected both black and white pixels; seen0=%v seen255=%v", seen0, seen255)
	}
}
