package stdimg

import (
	"image"
)

// AdaptiveResize resizes src to width x height using ResampleLanczos.
// If width or height is 0 the aspect ratio is preserved. If both are 0
// a clone of the source is returned. The parameter a controls the Lanczos
// window (commonly 3.0).
func AdaptiveResize(src *image.NRGBA, width, height int, a float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	sw := b.Dx()
	h := b.Dy()
	if width == 0 && height == 0 {
		return CloneNRGBA(src)
	}
	w := width
	hh := height
	if w == 0 {
		// preserve aspect
		w = int((float64(sw) * float64(hh)) / float64(h))
	}
	if hh == 0 {
		hh = int((float64(h) * float64(w)) / float64(sw))
	}
	if w <= 0 || hh <= 0 {
		return CloneNRGBA(src)
	}
	return ResampleLanczos(src, w, hh, a)
}
