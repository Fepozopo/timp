package stdimg

import (
	"image"
	"image/color"
	"testing"
)

func TestSepiaToneFull(t *testing.T) {
	// Create a 1x1 image with a known color
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.Pix[0] = 120 // r
	src.Pix[1] = 200 // g
	src.Pix[2] = 80  // b
	src.Pix[3] = 255 // a

	out := SepiaTone(src, 1.0)
	if out == nil {
		t.Fatal("SepiaTone returned nil")
	}
	// compute expected using the matrix
	r := float64(120)
	g := float64(200)
	b := float64(80)
	expR := 0.393*r + 0.769*g + 0.189*b
	expG := 0.349*r + 0.686*g + 0.168*b
	expB := 0.272*r + 0.534*g + 0.131*b
	if expR > 255 {
		expR = 255
	}
	if expG > 255 {
		expG = 255
	}
	if expB > 255 {
		expB = 255
	}

	i := out.PixOffset(0, 0)
	if out.Pix[i+0] != uint8(expR) || out.Pix[i+1] != uint8(expG) || out.Pix[i+2] != uint8(expB) {
		t.Fatalf("unexpected sepia pixel: got %v expected %v", out.Pix[i:i+3], []uint8{uint8(expR), uint8(expG), uint8(expB)})
	}
	if out.Pix[i+3] != 255 {
		t.Fatalf("alpha changed: %d", out.Pix[i+3])
	}
}

func TestSepiaToneBlend(t *testing.T) {
	// 2x1 image: left pixel red, right pixel blue
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	// left red
	src.Pix[0] = 255
	src.Pix[1] = 0
	src.Pix[2] = 0
	src.Pix[3] = 255
	// right blue
	src.Pix[4] = 0
	src.Pix[5] = 0
	src.Pix[6] = 255
	src.Pix[7] = 255

	out := SepiaTone(src, 0.5)
	if out == nil {
		t.Fatal("SepiaTone returned nil")
	}
	// verify values changed but are between original and full sepia
	for x := 0; x < 2; x++ {
		i := out.PixOffset(x, 0)
		r := src.Pix[i+0]
		g := src.Pix[i+1]
		b := src.Pix[i+2]
		// compute full sepia
		expR := 0.393*float64(r) + 0.769*float64(g) + 0.189*float64(b)
		expG := 0.349*float64(r) + 0.686*float64(g) + 0.168*float64(b)
		expB := 0.272*float64(r) + 0.534*float64(g) + 0.131*float64(b)
		if expR > 255 {
			expR = 255
		}
		if expG > 255 {
			expG = 255
		}
		if expB > 255 {
			expB = 255
		}
		// blended expected
		expBlendR := uint8((float64(r)*(1.0-0.5) + expR*0.5))
		expBlendG := uint8((float64(g)*(1.0-0.5) + expG*0.5))
		expBlendB := uint8((float64(b)*(1.0-0.5) + expB*0.5))
		if out.Pix[i+0] != expBlendR || out.Pix[i+1] != expBlendG || out.Pix[i+2] != expBlendB {
			t.Fatalf("pixel %d mismatch: got %v expected %v", x, out.Pix[i:i+3], []uint8{expBlendR, expBlendG, expBlendB})
		}
		if out.Pix[i+3] != 255 {
			t.Fatalf("alpha changed for pixel %d", x)
		}
	}
}

func TestSepiaEngineArgParsing(t *testing.T) {
	// simple 1x1 image
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{10, 20, 30, 255})
	img, err := ApplyCommandStdlib(src, "sepia", []string{"50%"})
	if err != nil {
		t.Fatalf("ApplyCommandStdlib sepia failed: %v", err)
	}
	if img == nil {
		t.Fatalf("img is nil")
	}
}
