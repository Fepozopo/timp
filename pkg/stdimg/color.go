package stdimg

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// RGB<->HSL conversions operate on 0..1 floats.

func rgbToHsl(r, g, b float64) (h, s, l float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2
	if max == min {
		// achromatic
		h = 0
		s = 0
		return
	}
	d := max - min
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func hslToRgb(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		// achromatic
		r = l
		g = l
		b = l
		return
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r = hueToRgb(p, q, h+1.0/3.0)
	g = hueToRgb(p, q, h)
	b = hueToRgb(p, q, h-1.0/3.0)
	return
}

// Modulate adjusts brightness (percent), saturation (percent), and hue (degrees).
// brightness and saturation are given as percentages where 100 means unchanged.
// hue is in degrees and will be added to the hue channel.
func Modulate(src *image.NRGBA, brightnessPct, saturationPct, hueDegrees float64) *image.NRGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewNRGBA(b)
	bFactor := brightnessPct / 100.0
	sFactor := saturationPct / 100.0
	hueShift := hueDegrees / 360.0 // convert to 0..1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x, y)
			r := float64(src.Pix[i+0]) / 255.0
			g := float64(src.Pix[i+1]) / 255.0
			b_ := float64(src.Pix[i+2]) / 255.0
			a := src.Pix[i+3]

			h, s, l := rgbToHsl(r, g, b_)
			// apply hue shift
			h = math.Mod(h+hueShift, 1.0)
			// adjust saturation and lightness
			s = clamp01(s * sFactor)
			l = clamp01(l * bFactor)
			r2, g2, b2 := hslToRgb(h, s, l)
			out.Pix[i+0] = uint8(clampFloatToUint8(r2 * 255.0))
			out.Pix[i+1] = uint8(clampFloatToUint8(g2 * 255.0))
			out.Pix[i+2] = uint8(clampFloatToUint8(b2 * 255.0))
			out.Pix[i+3] = a
		}
	}
	return out
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
