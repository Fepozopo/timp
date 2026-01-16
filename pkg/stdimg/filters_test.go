package stdimg

import (
	"image"
	"image/color"
	"testing"
)

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
