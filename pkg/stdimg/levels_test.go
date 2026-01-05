package stdimg

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
	"time"
)

func makeSolid(w, h int, c color.NRGBA) *image.NRGBA {
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

func TestMedianFilterSingleImpulse(t *testing.T) {
	// 5x5 image filled with gray; one impulse at center should be removed by median radius=1
	src := makeSolid(5, 5, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	// impulse
	i := src.PixOffset(2, 2)
	src.Pix[i+0] = 255
	src.Pix[i+1] = 255
	src.Pix[i+2] = 255

	out := MedianFilter(src, 1)
	if out == nil {
		t.Fatalf("MedianFilter returned nil")
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("output bounds mismatch")
	}
	// center should be restored to background (100)
	ci := out.PixOffset(2, 2)
	if out.Pix[ci+0] != 100 || out.Pix[ci+1] != 100 || out.Pix[ci+2] != 100 {
		t.Fatalf("expected center to be 100 after median, got R=%d G=%d B=%d", out.Pix[ci+0], out.Pix[ci+1], out.Pix[ci+2])
	}
}

func TestMedianFilterEdgesNoPanic(t *testing.T) {
	// small image where radius is larger than image half-size
	src := makeSolid(3, 3, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	out := MedianFilter(src, 5) // should not panic and should return an image
	if out == nil {
		t.Fatalf("MedianFilter returned nil for large radius")
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("output bounds mismatch for large radius")
	}
}

func BenchmarkMedianFilterRadius1(b *testing.B) {
	rand.Seed(42)
	w, h := 512, 512
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(src.Pix); i++ {
		src.Pix[i] = uint8(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MedianFilter(src, 1)
	}
}

func BenchmarkMedianFilterRadius3(b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	w, h := 512, 512
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(src.Pix); i++ {
		src.Pix[i] = uint8(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MedianFilter(src, 3)
	}
}
