package stdimg

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Annotate draws text onto the image at given position x,y (pixel coords) with given font size and color.
// fontPath may be empty to use a built-in basic font. Size is in points (if using TTF), ignored for basic font.
func Annotate(src *image.NRGBA, text string, fontPath string, size float64, x, y int, col color.Color) (*image.NRGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("source image is nil")
	}
	out := CloneNRGBA(src)
	var face font.Face
	if fontPath != "" {
		data, err := os.ReadFile(fontPath)
		if err != nil {
			log.Printf("failed to read font file %s: %v, falling back to basic font", fontPath, err)
			face = basicfont.Face7x13
		} else {
			tt, err := opentype.Parse(data)
			if err != nil {
				log.Printf("failed to parse font: %v, falling back to basic", err)
				face = basicfont.Face7x13
			} else {
				faceTmp, err := opentype.NewFace(tt, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
				if err != nil {
					log.Printf("failed to create font face: %v, falling back to basic", err)
					face = basicfont.Face7x13
				} else {
					face = faceTmp
				}
			}
		}
	} else {
		face = basicfont.Face7x13
	}

	d := &font.Drawer{
		Dst:  out,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
	return out, nil
}
