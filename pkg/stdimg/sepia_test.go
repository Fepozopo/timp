package stdimg

import (
	"image"
	"image/color"
	"math"
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
	// target sepia color
	tcol, _ := parseHexColor("#704214")
	var tNRGBA color.NRGBA
	if c, ok := tcol.(color.NRGBA); ok {
		tNRGBA = c
	} else {
		r, g, b, a := tcol.RGBA()
		tNRGBA = color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	}

	// get output pixel and compare in Lab space
	i := out.PixOffset(0, 0)
	outCol := color.NRGBA{out.Pix[i+0], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3]}

	d := labDistanceSq(outCol, tNRGBA)
	if d > 4.0 {
		t.Fatalf("output not close to target sepia in Lab (sqdist=%v)", d)
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
	// verify outputs moved toward target sepia in Lab space
	tcol, _ := parseHexColor("#704214")
	var tNRGBA color.NRGBA
	if c, ok := tcol.(color.NRGBA); ok {
		tNRGBA = c
	} else {
		r, g, b, a := tcol.RGBA()
		tNRGBA = color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	}

	for x := 0; x < 2; x++ {
		i := out.PixOffset(x, 0)
		orig := color.NRGBA{src.Pix[i+0], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3]}
		outCol := color.NRGBA{out.Pix[i+0], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3]}
		dOrigTarget := math.Sqrt(labDistanceSq(orig, tNRGBA))
		dOrigOut := math.Sqrt(labDistanceSq(orig, outCol))
		if dOrigOut <= 0.0 {
			t.Fatalf("output did not change for pixel %d", x)
		}
		if !(dOrigOut < dOrigTarget) {
			t.Fatalf("output did not move toward target for pixel %d (dOrigOut=%v dOrigTarget=%v)", x, dOrigOut, dOrigTarget)
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
