package stdimg

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

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
		data, err := ioutil.ReadFile(fontPath)
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

// parseColorString accepts multiple forms: named colors (basic set), #rrggbb, #rrggbbaa, #rgb, #rgba
func parseHexColor(s string) (color.Color, error) {
	if s == "" {
		return nil, fmt.Errorf("empty color")
	}
	s = strings.TrimSpace(s)
	// named colors (CSS Level 4 list)
	names := map[string]string{
		"aliceblue":            "#f0f8ff",
		"antiquewhite":         "#faebd7",
		"aqua":                 "#00ffff",
		"aquamarine":           "#7fffd4",
		"azure":                "#f0ffff",
		"beige":                "#f5f5dc",
		"bisque":               "#ffe4c4",
		"black":                "#000000",
		"blanchedalmond":       "#ffebcd",
		"blue":                 "#0000ff",
		"blueviolet":           "#8a2be2",
		"brown":                "#a52a2a",
		"burlywood":            "#deb887",
		"cadetblue":            "#5f9ea0",
		"chartreuse":           "#7fff00",
		"chocolate":            "#d2691e",
		"coral":                "#ff7f50",
		"cornflowerblue":       "#6495ed",
		"cornsilk":             "#fff8dc",
		"crimson":              "#dc143c",
		"cyan":                 "#00ffff",
		"darkblue":             "#00008b",
		"darkcyan":             "#008b8b",
		"darkgoldenrod":        "#b8860b",
		"darkgray":             "#a9a9a9",
		"darkgreen":            "#006400",
		"darkgrey":             "#a9a9a9",
		"darkkhaki":            "#bdb76b",
		"darkmagenta":          "#8b008b",
		"darkolivegreen":       "#556b2f",
		"darkorange":           "#ff8c00",
		"darkorchid":           "#9932cc",
		"darkred":              "#8b0000",
		"darksalmon":           "#e9967a",
		"darkseagreen":         "#8fbc8f",
		"darkslateblue":        "#483d8b",
		"darkslategray":        "#2f4f4f",
		"darkslategrey":        "#2f4f4f",
		"darkturquoise":        "#00ced1",
		"darkviolet":           "#9400d3",
		"deeppink":             "#ff1493",
		"deepskyblue":          "#00bfff",
		"dimgray":              "#696969",
		"dimgrey":              "#696969",
		"dodgerblue":           "#1e90ff",
		"firebrick":            "#b22222",
		"floralwhite":          "#fffaf0",
		"forestgreen":          "#228b22",
		"fuchsia":              "#ff00ff",
		"gainsboro":            "#dcdcdc",
		"ghostwhite":           "#f8f8ff",
		"goldenrod":            "#daa520",
		"gold":                 "#ffd700",
		"gray":                 "#808080",
		"green":                "#008000",
		"greenyellow":          "#adff2f",
		"grey":                 "#808080",
		"honeydew":             "#f0fff0",
		"hotpink":              "#ff69b4",
		"indianred":            "#cd5c5c",
		"indigo":               "#4b0082",
		"ivory":                "#fffff0",
		"khaki":                "#f0e68c",
		"lavenderblush":        "#fff0f5",
		"lavender":             "#e6e6fa",
		"lawngreen":            "#7cfc00",
		"lemonchiffon":         "#fffacd",
		"lightblue":            "#add8e6",
		"lightcoral":           "#f08080",
		"lightcyan":            "#e0ffff",
		"lightgoldenrodyellow": "#fafad2",
		"lightgray":            "#d3d3d3",
		"lightgreen":           "#90ee90",
		"lightgrey":            "#d3d3d3",
		"lightpink":            "#ffb6c1",
		"lightsalmon":          "#ffa07a",
		"lightseagreen":        "#20b2aa",
		"lightskyblue":         "#87cefa",
		"lightslategray":       "#778899",
		"lightslategrey":       "#778899",
		"lightsteelblue":       "#b0c4de",
		"lightyellow":          "#ffffe0",
		"lime":                 "#00ff00",
		"limegreen":            "#32cd32",
		"linen":                "#faf0e6",
		"magenta":              "#ff00ff",
		"maroon":               "#800000",
		"mediumaquamarine":     "#66cdaa",
		"mediumblue":           "#0000cd",
		"mediumorchid":         "#ba55d3",
		"mediumpurple":         "#9370db",
		"mediumseagreen":       "#3cb371",
		"mediumslateblue":      "#7b68ee",
		"mediumspringgreen":    "#00fa9a",
		"mediumturquoise":      "#48d1cc",
		"mediumvioletred":      "#c71585",
		"midnightblue":         "#191970",
		"mintcream":            "#f5fffa",
		"mistyrose":            "#ffe4e1",
		"moccasin":             "#ffe4b5",
		"navajowhite":          "#ffdead",
		"navy":                 "#000080",
		"oldlace":              "#fdf5e6",
		"olive":                "#808000",
		"olivedrab":            "#6b8e23",
		"orange":               "#ffa500",
		"orangered":            "#ff4500",
		"orchid":               "#da70d6",
		"palegoldenrod":        "#eee8aa",
		"palegreen":            "#98fb98",
		"paleturquoise":        "#afeeee",
		"palevioletred":        "#db7093",
		"papayawhip":           "#ffefd5",
		"peachpuff":            "#ffdab9",
		"peru":                 "#cd853f",
		"pink":                 "#ffc0cb",
		"plum":                 "#dda0dd",
		"powderblue":           "#b0e0e6",
		"purple":               "#800080",
		"rebeccapurple":        "#663399",
		"red":                  "#ff0000",
		"rosybrown":            "#bc8f8f",
		"royalblue":            "#4169e1",
		"saddlebrown":          "#8b4513",
		"salmon":               "#fa8072",
		"sandybrown":           "#f4a460",
		"seagreen":             "#2e8b57",
		"seashell":             "#fff5ee",
		"sienna":               "#a0522d",
		"silver":               "#c0c0c0",
		"skyblue":              "#87ceeb",
		"slateblue":            "#6a5acd",
		"slategray":            "#708090",
		"slategrey":            "#708090",
		"snow":                 "#fffafa",
		"springgreen":          "#00ff7f",
		"steelblue":            "#4682b4",
		"tan":                  "#d2b48c",
		"teal":                 "#008080",
		"thistle":              "#d8bfd8",
		"tomato":               "#ff6347",
		"turquoise":            "#40e0d0",
		"violet":               "#ee82ee",
		"wheat":                "#f5deb3",
		"white":                "#ffffff",
		"whitesmoke":           "#f5f5f5",
		"yellow":               "#ffff00",
		"yellowgreen":          "#9acd32",
	}
	if hexs, ok := names[strings.ToLower(s)]; ok {
		return parseHexColor(hexs)
	}
	if s[0] != '#' {
		return nil, fmt.Errorf("unsupported color format: %s", s)
	}
	hex := s[1:]
	var r, g, b, a uint8
	switch len(hex) {
	case 3: // #rgb
		rh, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		gh, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		bh, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		r = uint8(rh)
		g = uint8(gh)
		b = uint8(bh)
		a = 0xff
	case 4: // #rgba
		rh, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		gh, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		bh, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		ah, _ := strconv.ParseUint(string(hex[3])+string(hex[3]), 16, 8)
		r = uint8(rh)
		g = uint8(gh)
		b = uint8(bh)
		a = uint8(ah)
	case 6: // #rrggbb
		rh, err := strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return nil, err
		}
		gh, err := strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return nil, err
		}
		bh, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return nil, err
		}
		r = uint8(rh)
		g = uint8(gh)
		b = uint8(bh)
		a = 0xff
	case 8: // #rrggbbaa
		rh, err := strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return nil, err
		}
		gh, err := strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return nil, err
		}
		bh, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return nil, err
		}
		ah, err := strconv.ParseUint(hex[6:8], 16, 8)
		if err != nil {
			return nil, err
		}
		r = uint8(rh)
		g = uint8(gh)
		b = uint8(bh)
		a = uint8(ah)
	default:
		return nil, fmt.Errorf("unsupported hex color length: %d", len(hex))
	}
	return color.NRGBA{r, g, b, a}, nil
}
