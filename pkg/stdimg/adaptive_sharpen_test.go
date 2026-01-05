package stdimg

import (
	"image"
	"testing"
)

func TestAdaptiveSharpenBasic(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	// draw a simple edge
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			i := src.PixOffset(x, y)
			if x < 4 {
				src.Pix[i+0] = 0
				src.Pix[i+1] = 0
				src.Pix[i+2] = 0
			} else {
				src.Pix[i+0] = 255
				src.Pix[i+1] = 255
				src.Pix[i+2] = 255
			}
			src.Pix[i+3] = 255
		}
	}
	out := AdaptiveSharpen(src, 0.0, 1.0, 1.0)
	if out == nil {
		t.Fatal("output is nil")
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("bounds changed: %v vs %v", out.Bounds(), src.Bounds())
	}
}
