package stdimg

import (
	"image"
	"image/color"
	"testing"
)

func TestFloodfillPaintSimple(t *testing.T) {
	// 5x5 image: center 3x3 region is red, border is blue
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	red := color.NRGBA{255, 0, 0, 255}
	blue := color.NRGBA{0, 0, 255, 255}
	// fill with blue by default
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.Pix[img.PixOffset(x, y)+0] = blue.R
			img.Pix[img.PixOffset(x, y)+1] = blue.G
			img.Pix[img.PixOffset(x, y)+2] = blue.B
			img.Pix[img.PixOffset(x, y)+3] = blue.A
		}
	}
	// make center 3x3 red
	for y := 1; y <= 3; y++ {
		for x := 1; x <= 3; x++ {
			off := img.PixOffset(x, y)
			img.Pix[off+0] = red.R
			img.Pix[off+1] = red.G
			img.Pix[off+2] = red.B
			img.Pix[off+3] = red.A
		}
	}
	// floodfill from (2,2) with green
	green := color.NRGBA{0, 255, 0, 255}
	out := FloodfillPaint(img, green, 0.0, color.NRGBA{0, 0, 0, 0}, 2, 2, false)
	// check that center 3x3 are green and border remains blue
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			off := out.PixOffset(x, y)
			c := color.NRGBA{out.Pix[off+0], out.Pix[off+1], out.Pix[off+2], out.Pix[off+3]}
			if x >= 1 && x <= 3 && y >= 1 && y <= 3 {
				if c != green {
					t.Fatalf("expected green at %d,%d got %v", x, y, c)
				}
			} else {
				if c != blue {
					t.Fatalf("expected blue at %d,%d got %v", x, y, c)
				}
			}
		}
	}
}

func TestFloodfillPaintBorder(t *testing.T) {
	// 5x5 white image with black border forming a box; fill inside using border color
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	white := color.NRGBA{255, 255, 255, 255}
	black := color.NRGBA{0, 0, 0, 255}
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			off := img.PixOffset(x, y)
			img.Pix[off+0] = white.R
			img.Pix[off+1] = white.G
			img.Pix[off+2] = white.B
			img.Pix[off+3] = white.A
		}
	}
	// draw a black border at x==0, x==4, y==0, y==4
	for i := 0; i < 5; i++ {
		off := img.PixOffset(i, 0)
		img.Pix[off+0] = black.R
		img.Pix[off+1] = black.G
		img.Pix[off+2] = black.B
		img.Pix[off+3] = black.A
		off = img.PixOffset(i, 4)
		img.Pix[off+0] = black.R
		img.Pix[off+1] = black.G
		img.Pix[off+2] = black.B
		img.Pix[off+3] = black.A
		off = img.PixOffset(0, i)
		img.Pix[off+0] = black.R
		img.Pix[off+1] = black.G
		img.Pix[off+2] = black.B
		img.Pix[off+3] = black.A
		off = img.PixOffset(4, i)
		img.Pix[off+0] = black.R
		img.Pix[off+1] = black.G
		img.Pix[off+2] = black.B
		img.Pix[off+3] = black.A
	}
	// fill from inside (2,2) using border color detection
	fillCol := color.NRGBA{255, 0, 0, 255}
	out := FloodfillPaint(img, fillCol, 0.0, black, 2, 2, false)
	// interior (1..3,1..3) should be filled red, border remains black
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			off := out.PixOffset(x, y)
			c := color.NRGBA{out.Pix[off+0], out.Pix[off+1], out.Pix[off+2], out.Pix[off+3]}
			if x == 0 || y == 0 || x == 4 || y == 4 {
				if c != black {
					t.Fatalf("expected black border at %d,%d got %v", x, y, c)
				}
			} else {
				if c != fillCol {
					t.Fatalf("expected filled red at %d,%d got %v", x, y, c)
				}
			}
		}
	}
}

func TestFloodfillPaintInvert(t *testing.T) {
	// 3x1 image: [A, B, A]; invert should fill B when starting at A
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	A := color.NRGBA{10, 20, 30, 255}
	B := color.NRGBA{200, 210, 220, 255}
	for x := 0; x < 3; x++ {
		off := img.PixOffset(x, 0)
		if x == 1 {
			img.Pix[off+0] = B.R
			img.Pix[off+1] = B.G
			img.Pix[off+2] = B.B
			img.Pix[off+3] = B.A
		} else {
			img.Pix[off+0] = A.R
			img.Pix[off+1] = A.G
			img.Pix[off+2] = A.B
			img.Pix[off+3] = A.A
		}
	}
	// start at 0 (A), invert true, fill with green
	green := color.NRGBA{0, 255, 0, 255}
	out := FloodfillPaint(img, green, 0.0, color.NRGBA{0, 0, 0, 0}, 0, 0, true)
	// expect positions 0 and 2 remain A, position 1 becomes green
	c0 := color.NRGBA{out.Pix[0], out.Pix[1], out.Pix[2], out.Pix[3]}
	c1 := color.NRGBA{out.Pix[out.PixOffset(1, 0)+0], out.Pix[out.PixOffset(1, 0)+1], out.Pix[out.PixOffset(1, 0)+2], out.Pix[out.PixOffset(1, 0)+3]}
	c2 := color.NRGBA{out.Pix[out.PixOffset(2, 0)+0], out.Pix[out.PixOffset(2, 0)+1], out.Pix[out.PixOffset(2, 0)+2], out.Pix[out.PixOffset(2, 0)+3]}
	if c1 != green {
		t.Fatalf("expected middle pixel green, got %v", c1)
	}
	if c0 != A || c2 != A {
		t.Fatalf("expected edge pixels unchanged: %v %v", c0, c2)
	}
}
