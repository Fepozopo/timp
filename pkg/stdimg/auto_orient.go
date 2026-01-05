package stdimg

import (
	"image"
)

// AutoOrient applies EXIF orientation to an image.Image and returns a new image.Image.
// orientation follows EXIF spec (1..8). If orientation is 1 or unknown, the image is returned as-is.
func AutoOrient(img image.Image, orientation int) image.Image {
	if img == nil {
		return nil
	}
	if orientation <= 1 || orientation > 8 {
		return img
	}
	src := ToNRGBA(img)
	switch orientation {
	case 2:
		return FlopNRGBA(src)
	case 3:
		return Rotate180NRGBA(src)
	case 4:
		return FlipNRGBA(src)
	case 5:
		// transpose: rotate 90 CW then flip horizontal
		tmp := Rotate90CWNRGBA(src)
		return FlopNRGBA(ToNRGBA(tmp))
	case 6:
		return Rotate90CWNRGBA(src)
	case 7:
		// transverse: rotate 90 CCW then flip horizontal
		tmp := Rotate90CCWNRGBA(src)
		return FlopNRGBA(ToNRGBA(tmp))
	case 8:
		return Rotate90CCWNRGBA(src)
	default:
		return img
	}
}

// helper transforms operating on *image.NRGBA
func FlipNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIdx := src.PixOffset(x, y)
			dstIdx := out.PixOffset(x, h-1-y)
			copy(out.Pix[dstIdx:dstIdx+4], src.Pix[srcIdx:srcIdx+4])
		}
	}
	return out
}

func FlopNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIdx := src.PixOffset(x, y)
			dstIdx := out.PixOffset(w-1-x, y)
			copy(out.Pix[dstIdx:dstIdx+4], src.Pix[srcIdx:srcIdx+4])
		}
	}
	return out
}

func Rotate180NRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIdx := src.PixOffset(x, y)
			dstIdx := out.PixOffset(w-1-x, h-1-y)
			copy(out.Pix[dstIdx:dstIdx+4], src.Pix[srcIdx:srcIdx+4])
		}
	}
	return out
}

func Rotate90CWNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIdx := src.PixOffset(x, y)
			dstIdx := out.PixOffset(h-1-y, x)
			copy(out.Pix[dstIdx:dstIdx+4], src.Pix[srcIdx:srcIdx+4])
		}
	}
	return out
}

func Rotate90CCWNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcIdx := src.PixOffset(x, y)
			dstIdx := out.PixOffset(y, w-1-x)
			copy(out.Pix[dstIdx:dstIdx+4], src.Pix[srcIdx:srcIdx+4])
		}
	}
	return out
}
