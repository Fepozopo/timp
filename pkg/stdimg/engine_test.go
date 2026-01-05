package stdimg

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func makeSolidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i+0] = c.R
			img.Pix[i+1] = c.G
			img.Pix[i+2] = c.B
			img.Pix[i+3] = c.A
		}
	}
	return img
}

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

func TestDespeckleEngine(t *testing.T) {
	// create small image with speckles
	src := makeSolidNRGBA(7, 7, color.NRGBA{R: 120, G: 120, B: 120, A: 255})
	// add speckles
	src.Pix[src.PixOffset(3, 1)+0] = 255
	src.Pix[src.PixOffset(1, 4)+1] = 255
	src.Pix[src.PixOffset(5, 5)+2] = 255

	outImg, err := ApplyCommandStdlib(src, "despeckle", []string{"1"})
	if err != nil {
		t.Fatalf("despeckle command failed: %v", err)
	}
	out, ok := outImg.(*image.NRGBA)
	if !ok {
		t.Fatalf("expected *image.NRGBA output from despeckle")
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("despeckle output bounds mismatch")
	}

	// Now test edge command on a simple pattern
	pat := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	// draw a black cross on white
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			i := pat.PixOffset(x, y)
			pat.Pix[i+0] = 255
			pat.Pix[i+1] = 255
			pat.Pix[i+2] = 255
			pat.Pix[i+3] = 255
		}
	}
	// vertical line
	for y := 0; y < 5; y++ {
		i := pat.PixOffset(2, y)
		pat.Pix[i+0] = 0
		pat.Pix[i+1] = 0
		pat.Pix[i+2] = 0
	}
	out2, err := ApplyCommandStdlib(pat, "edge", []string{"1.0"})
	if err != nil {
		t.Fatalf("edge command failed: %v", err)
	}
	outEdge, ok := out2.(*image.NRGBA)
	if !ok {
		t.Fatalf("expected *image.NRGBA output from edge")
	}
	// ensure edge detects the vertical line by checking adjacent columns have non-zero edge values
	leftIdx := outEdge.PixOffset(1, 2)
	rightIdx := outEdge.PixOffset(3, 2)
	if outEdge.Pix[leftIdx+0] == 0 && outEdge.Pix[rightIdx+0] == 0 {
		t.Fatalf("edge did not detect expected line (both adjacent columns are zero)")
	}
}
