package stdimg

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestCompositeBasic(t *testing.T) {
	bg := makeSolidNRGBA(80, 60, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	fg := makeSolidNRGBA(20, 20, color.NRGBA{R: 0, G: 0, B: 255, A: 128})
	// write fg to a temp file
	f, err := os.CreateTemp("", "fg-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	png.Encode(f, fg)
	f.Close()

	outImg, err := ApplyCommandStdlib(bg, "composite", []string{f.Name(), "OVER", "10", "5"})
	if err != nil {
		t.Fatalf("composite failed: %v", err)
	}
	out, ok := outImg.(*image.NRGBA)
	if !ok {
		t.Fatalf("expected *image.NRGBA output")
	}
	// check a pixel in the composite area to ensure blending changed it
	idx := out.PixOffset(12, 7)
	r := out.Pix[idx+0]
	g := out.Pix[idx+1]
	b := out.Pix[idx+2]
	if r == 255 && g == 0 && b == 0 {
		t.Fatalf("expected composite to modify pixel, got pure background")
	}
	// save for inspection optionally
	if os.Getenv("TIMP_SAVE_TEST_OUTPUT") == "1" {
		f2, _ := os.Create("composite_test_out.png")
		defer f2.Close()
		png.Encode(f2, out)
	}
}
