package stdimg

import (
	"image"
	"testing"
)

func TestAdaptiveResizeBasic(t *testing.T) {
	// create a simple 8x4 image
	src := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	// fill with a gradient
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			i := src.PixOffset(x, y)
			src.Pix[i+0] = uint8(x * 32)
			src.Pix[i+1] = uint8(y * 64)
			src.Pix[i+2] = 0
			src.Pix[i+3] = 255
		}
	}
	out := AdaptiveResize(src, 4, 2, 3.0)
	if out == nil {
		t.Fatal("output is nil")
	}
	if out.Bounds().Dx() != 4 || out.Bounds().Dy() != 2 {
		t.Fatalf("unexpected size: %v", out.Bounds())
	}
	// upscale
	out2 := AdaptiveResize(src, 16, 8, 3.0)
	if out2 == nil {
		t.Fatal("output2 is nil")
	}
	if out2.Bounds().Dx() != 16 || out2.Bounds().Dy() != 8 {
		t.Fatalf("unexpected size up: %v", out2.Bounds())
	}
	// height only
	out3 := AdaptiveResize(src, 0, 200, 3.0)
	if out3 == nil {
		t.Fatal("output3 is nil")
	}
	if out3.Bounds().Dy() != 200 {
		t.Fatalf("unexpected height: %d", out3.Bounds().Dy())
	}
}
