package stdimg

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestAnnotateBasic(t *testing.T) {
	src := makeSolidNRGBA(100, 50, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	out, err := ApplyCommandStdlib(src, "annotate", []string{"Hello", "12", "10", "20", "#000000"})
	if err != nil {
		t.Fatalf("annotate failed: %v", err)
	}
	if out == nil {
		t.Fatalf("annotate returned nil image")
	}
	// save to tmp for manual inspection if env var set
	if os.Getenv("TIMP_SAVE_TEST_OUTPUT") == "1" {
		f, _ := os.Create("annotate_test_out.png")
		defer f.Close()
		png.Encode(f, out)
	}
}

func TestAnnotateWithFontFile(t *testing.T) {
	// This test will only run if a font file path is provided via env var TIMP_TEST_FONT
	fontPath := os.Getenv("TIMP_TEST_FONT")
	if fontPath == "" {
		t.Skip("no font provided")
	}
	bg := makeSolidNRGBA(200, 50, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	outImg, err := ApplyCommandStdlib(bg, "annotate", []string{"HelloWorld", fontPath, "24", "10", "30", "#ff0000"})
	if err != nil {
		t.Fatalf("annotate with font failed: %v", err)
	}
	if outImg == nil {
		t.Fatalf("annotate returned nil image")
	}
	out, ok := outImg.(*image.NRGBA)
	if !ok {
		t.Fatalf("expected *image.NRGBA output from annotate")
	}
	// ensure some pixel changed from white
	okChanged := false
	b := out.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !okChanged; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := out.PixOffset(x, y)
			if out.Pix[i+0] != 255 || out.Pix[i+1] != 255 || out.Pix[i+2] != 255 {
				okChanged = true
				break
			}
		}
	}
	if !okChanged {
		t.Fatalf("expected annotate to draw non-white pixels")
	}
}
