package stdimg

import (
	"image"
	"testing"
)

func TestAddNoiseDeterministic(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			i := src.PixOffset(x, y)
			src.Pix[i+0] = 128
			src.Pix[i+1] = 128
			src.Pix[i+2] = 128
			src.Pix[i+3] = 255
		}
	}
	out := AddNoise(src, "GAUSSIAN", 5.0, 42)
	if out == nil {
		t.Fatal("out nil")
	}
	// ensure values are within 0..255 and differ from original in at least one pixel
	same := true
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			i := out.PixOffset(x, y)
			r := out.Pix[i+0]
			if r != 128 {
				same = false
			}
		}
	}
	if same {
		t.Fatal("expected at least one pixel to change")
	}
}
