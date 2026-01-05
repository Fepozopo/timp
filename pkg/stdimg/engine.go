package stdimg

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"strconv"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// ApplyCommandStdlib applies basic commands to an image.NRGBA and returns a new image.
// It implements a subset of the original ImageMagick-backed commands: resize, rotate, blur, sharpen, crop, flip, flop, grayscale, strip, identify (returns nil image and prints info).
func ApplyCommandStdlib(img image.Image, commandName string, args []string) (image.Image, error) {
	if img == nil {
		return nil, fmt.Errorf("source image is nil")
	}
	src := ToNRGBA(img)
	switch commandName {
	case "resize":
		if len(args) != 2 {
			return nil, fmt.Errorf("resize requires 2 args: width height")
		}
		w, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}
		h, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, fmt.Errorf("invalid height: %w", err)
		}
		// use Lanczos a=3
		out := ResampleLanczos(src, w, h, 3.0)
		return out, nil

	case "rotate":
		if len(args) != 1 {
			return nil, fmt.Errorf("rotate requires 1 arg: degrees")
		}
		deg, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid degrees: %w", err)
		}
		// do a simple rotate using inverse mapping with bilinear sampling
		rad := deg * (math.Pi / 180.0)
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		// compute new bounds
		w0 := src.Bounds().Dx()
		h0 := src.Bounds().Dy()
		// compute corners
		cx := float64(w0) / 2.0
		cy := float64(h0) / 2.0
		// approximate new bounds by rotating corners
		var xs [4]float64
		var ys [4]float64
		corners := [4][2]float64{{0 - cx, 0 - cy}, {float64(w0) - cx, 0 - cy}, {float64(w0) - cx, float64(h0) - cy}, {0 - cx, float64(h0) - cy}}
		for i := 0; i < 4; i++ {
			xs[i] = corners[i][0]*cos - corners[i][1]*sin
			ys[i] = corners[i][0]*sin + corners[i][1]*cos
		}
		minX, maxX := xs[0], xs[0]
		minY, maxY := ys[0], ys[0]
		for i := 1; i < 4; i++ {
			if xs[i] < minX {
				minX = xs[i]
			}
			if xs[i] > maxX {
				maxX = xs[i]
			}
			if ys[i] < minY {
				minY = ys[i]
			}
			if ys[i] > maxY {
				maxY = ys[i]
			}
		}
		newW := int(math.Ceil(maxX - minX))
		newH := int(math.Ceil(maxY - minY))
		out := image.NewNRGBA(image.Rect(0, 0, newW, newH))
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				// map dest pixel to source coordinate
				xRel := float64(x) + minX
				yRel := float64(y) + minY
				sx := xRel*cos + yRel*sin + cx
				sy := -xRel*sin + yRel*cos + cy
				rf, gf, bf, af := sampleBilinear(src, sx, sy)
				i := out.PixOffset(x, y)
				out.Pix[i+0] = uint8(clampFloatToUint8(rf))
				out.Pix[i+1] = uint8(clampFloatToUint8(gf))
				out.Pix[i+2] = uint8(clampFloatToUint8(bf))
				out.Pix[i+3] = uint8(clampFloatToUint8(af))
			}
		}
		return out, nil

	case "blur":
		// accept one arg: sigma
		if len(args) < 1 {
			return nil, fmt.Errorf("blur requires 1 arg: sigma")
		}
		sigma, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid sigma: %w", err)
		}
		out := SeparableGaussianBlur(src, sigma)
		return out, nil

	case "medianFilter":
		// medianFilter requires 1 arg: radius
		if len(args) != 1 {
			return nil, fmt.Errorf("medianFilter requires 1 arg: radius")
		}
		radius, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, fmt.Errorf("invalid radius: %w", err)
		}
		out := MedianFilter(src, radius)
		return out, nil

	case "despeckle":
		// despeckle [radius]
		radius := 1
		if len(args) >= 1 && args[0] != "" {
			if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
				radius = v
			}
		}
		out := Despeckle(src, radius)
		return out, nil

	case "level":
		// level requires 3 args: blackPoint gamma whitePoint
		if len(args) != 3 {
			return nil, fmt.Errorf("level requires 3 args: blackPoint gamma whitePoint")
		}
		blackPoint, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid blackPoint: %w", err)
		}
		gamma, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gamma: %w", err)
		}
		whitePoint, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid whitePoint: %w", err)
		}
		out := Level(src, blackPoint, gamma, whitePoint)
		return out, nil

	case "normalize":
		// normalize takes no args
		if len(args) != 0 {
			return nil, fmt.Errorf("normalize takes no args")
		}
		out := Normalize(src)
		return out, nil

	case "autoLevel":
		// autoLevel takes no args
		if len(args) != 0 {
			return nil, fmt.Errorf("autoLevel takes no args")
		}
		out := AutoLevel(src)
		return out, nil

	case "autoGamma":
		// autoGamma takes no args
		if len(args) != 0 {
			return nil, fmt.Errorf("autoGamma takes no args")
		}
		out := AutoGamma(src)
		return out, nil

	case "gamma":
		// gamma requires 1 arg: gamma value
		if len(args) != 1 {
			return nil, fmt.Errorf("gamma requires 1 arg: gamma")
		}
		gammaVal, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gamma: %w", err)
		}
		out := Gamma(src, gammaVal)
		return out, nil

	case "negate":
		// negate [onlyGray]
		onlyGray := false
		if len(args) >= 1 && args[0] != "" {
			b, err := strconv.ParseBool(args[0])
			if err != nil {
				return nil, fmt.Errorf("invalid onlyGray flag: %w", err)
			}
			onlyGray = b
		}
		out := Negate(src, onlyGray)
		return out, nil

	case "threshold":
		// threshold <value> [perChannel]
		if len(args) < 1 {
			return nil, fmt.Errorf("threshold requires at least 1 arg: value")
		}
		threshVal, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid threshold value: %w", err)
		}
		perChannel := false
		if len(args) >= 2 && args[1] != "" {
			b, err := strconv.ParseBool(args[1])
			if err != nil {
				return nil, fmt.Errorf("invalid perChannel flag: %w", err)
			}
			perChannel = b
		}
		out := Threshold(src, threshVal, perChannel)
		return out, nil

	case "modulate":
		// modulate requires 3 args: brightness percent, saturation percent, hue degrees
		if len(args) != 3 {
			return nil, fmt.Errorf("modulate requires 3 args: brightness saturation hue")
		}
		brightness, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid brightness: %w", err)
		}
		saturation, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid saturation: %w", err)
		}
		hue, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid hue: %w", err)
		}
		return Modulate(src, brightness, saturation, hue), nil

	case "vignette":
		// vignette requires 4 or 5 args: radius sigma x y [strength]
		if len(args) < 4 {
			return nil, fmt.Errorf("vignette requires 4 args: radius sigma x y [strength]")
		}
		radius, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid radius: %w", err)
		}
		sigma, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid sigma: %w", err)
		}
		x, err := strconv.Atoi(args[2])
		if err != nil {
			return nil, fmt.Errorf("invalid x: %w", err)
		}
		y, err := strconv.Atoi(args[3])
		if err != nil {
			return nil, fmt.Errorf("invalid y: %w", err)
		}
		strength := 1.0
		if len(args) >= 5 && args[4] != "" {
			// support percent like "50%" or fraction like "0.5"
			if args[4][len(args[4])-1] == '%' {
				v, err := strconv.ParseFloat(args[4][:len(args[4])-1], 64)
				if err != nil {
					return nil, fmt.Errorf("invalid strength percent: %w", err)
				}
				strength = v / 100.0
			} else {
				v, err := strconv.ParseFloat(args[4], 64)
				if err != nil {
					return nil, fmt.Errorf("invalid strength: %w", err)
				}
				strength = v
			}
			if strength < 0 {
				strength = 0
			}
			if strength > 1 {
				strength = 1
			}
		}
		out := Vignette(src, radius, sigma, x, y, strength)
		return out, nil

	case "grayscale":
		// simple luminance conversion
		b := src.Bounds()
		out := image.NewNRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				i := src.PixOffset(x, y)
				r := src.Pix[i+0]
				g := src.Pix[i+1]
				b_ := src.Pix[i+2]
				a := src.Pix[i+3]
				// Rec. 709 luminance
				lum := uint8((0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b_)))
				out.Pix[i+0] = lum
				out.Pix[i+1] = lum
				out.Pix[i+2] = lum
				out.Pix[i+3] = a
			}
		}
		return out, nil

	case "edge":
		// edge [sigma] [scale] [threshold] [binary]
		// examples: "edge 0.0 1.0 0.0 false"
		sigma := 0.0
		scale := 1.0
		threshold := 0.0
		binary := false
		if len(args) >= 1 && args[0] != "" {
			if v, err := strconv.ParseFloat(args[0], 64); err == nil {
				sigma = v
			}
		}
		if len(args) >= 2 && args[1] != "" {
			if v, err := strconv.ParseFloat(args[1], 64); err == nil {
				scale = v
			}
		}
		if len(args) >= 3 && args[2] != "" {
			if v, err := strconv.ParseFloat(args[2], 64); err == nil {
				threshold = v
			}
		}
		if len(args) >= 4 && args[3] != "" {
			if b, err := strconv.ParseBool(args[3]); err == nil {
				binary = b
			}
		}
		out := EdgeEx(src, sigma, scale, threshold, binary)
		return out, nil

	case "adaptiveBlur":
		// adaptiveBlur [radius] [sigmaMin] [sigmaMax] [levels]
		// defaults: radius=1.0, sigmaMin=0.5, sigmaMax=1.0, levels=6
		radius := 1.0
		sigmaMin := 0.5
		sigmaMax := 1.0
		levels := 6
		if len(args) >= 1 && args[0] != "" {
			if v, err := strconv.ParseFloat(args[0], 64); err == nil {
				radius = v
			}
		}
		if len(args) >= 2 && args[1] != "" {
			if v, err := strconv.ParseFloat(args[1], 64); err == nil {
				sigmaMin = v
			}
		}
		if len(args) >= 3 && args[2] != "" {
			if v, err := strconv.ParseFloat(args[2], 64); err == nil {
				sigmaMax = v
			}
		}
		if len(args) >= 4 && args[3] != "" {
			if v, err := strconv.Atoi(args[3]); err == nil && v > 0 {
				levels = v
			}
		}
		out := AdaptiveBlurPerPixel(src, radius, sigmaMin, sigmaMax, levels)
		return out, nil

	case "adaptiveResize":
		// adaptiveResize [width] [height] [a]
		width := 0
		height := 0
		a := 3.0
		if len(args) >= 1 && args[0] != "" {
			if v, err := strconv.Atoi(args[0]); err == nil {
				width = v
			}
		}
		if len(args) >= 2 && args[1] != "" {
			if v, err := strconv.Atoi(args[1]); err == nil {
				height = v
			}
		}
		if len(args) >= 3 && args[2] != "" {
			if v, err := strconv.ParseFloat(args[2], 64); err == nil {
				a = v
			}
		}
		out := AdaptiveResize(src, width, height, a)
		return out, nil

	case "adaptiveSharpen":
		// adaptiveSharpen [radius] [sigma] [amount]
		radius := 0.0
		sigma := 1.0
		amount := 1.0
		if len(args) >= 1 && args[0] != "" {
			if v, err := strconv.ParseFloat(args[0], 64); err == nil {
				radius = v
			}
		}
		if len(args) >= 2 && args[1] != "" {
			if v, err := strconv.ParseFloat(args[1], 64); err == nil {
				sigma = v
			}
		}
		if len(args) >= 3 && args[2] != "" {
			if v, err := strconv.ParseFloat(args[2], 64); err == nil {
				amount = v
			}
		}
		out := AdaptiveSharpen(src, radius, sigma, amount)
		return out, nil

	case "adaptiveThreshold":
		// adaptiveThreshold [window_width] [window_height] [offset]
		ww := 15
		wh := 15
		off := 0.0
		if len(args) >= 1 && args[0] != "" {
			if v, err := strconv.Atoi(args[0]); err == nil {
				ww = v
			}
		}
		if len(args) >= 2 && args[1] != "" {
			if v, err := strconv.Atoi(args[1]); err == nil {
				wh = v
			}
		}
		if len(args) >= 3 && args[2] != "" {
			if v, err := strconv.ParseFloat(args[2], 64); err == nil {
				off = v
			}
		}
		out := AdaptiveThreshold(src, ww, wh, off)
		return out, nil

	case "addNoise":
		// addNoise [type] [amount] [seed]
		typ := "GAUSSIAN"
		amt := 10.0
		seed := int64(0)
		if len(args) >= 1 && args[0] != "" {
			typ = args[0]
		}
		if len(args) >= 2 && args[1] != "" {
			if v, err := strconv.ParseFloat(args[1], 64); err == nil {
				amt = v
			}
		}
		if len(args) >= 3 && args[2] != "" {
			if v, err := strconv.ParseInt(args[2], 10, 64); err == nil {
				seed = v
			}
		}
		out := AddNoise(src, typ, amt, seed)
		return out, nil

	case "crop":
		if len(args) != 4 {
			return nil, fmt.Errorf("crop requires 4 args: width height x y")
		}
		w, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}
		h, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, fmt.Errorf("invalid height: %w", err)
		}
		x0, err := strconv.Atoi(args[2])
		if err != nil {
			return nil, fmt.Errorf("invalid x: %w", err)
		}
		y0, err := strconv.Atoi(args[3])
		if err != nil {
			return nil, fmt.Errorf("invalid y: %w", err)
		}
		rect := image.Rect(x0, y0, x0+w, y0+h).Intersect(src.Bounds())
		out := image.NewNRGBA(rect)
		draw.Draw(out, rect.Sub(rect.Min), src, rect.Min, draw.Src)
		return out, nil

	case "flip":
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
		return out, nil

	case "flop":
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
		return out, nil

	case "histogram":
		// optional arg: bins
		bins := 256
		if len(args) > 0 && args[0] != "" {
			if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
				bins = v
			}
		}
		rHist, gHist, bHist := ComputeHistogram(src, bins)
		// Render a small histogram PNG and return as image
		histImg := RenderHistogramImage(rHist, gHist, bHist, 512, 120)
		return histImg, nil

	case "equalize":
		out := Equalize(src)
		return out, nil

	case "trim":
		// trim requires 1 arg: fuzz
		if len(args) < 1 {
			return nil, fmt.Errorf("trim requires 1 arg: fuzz")
		}
		// support percent or numeric
		fuzzStr := args[0]
		fuzz := 0.0
		if len(fuzzStr) > 0 && fuzzStr[len(fuzzStr)-1] == '%' {
			v, err := strconv.ParseFloat(fuzzStr[:len(fuzzStr)-1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid fuzz percent: %w", err)
			}
			fuzz = v * 255.0 / 100.0
		} else {
			v, err := strconv.ParseFloat(fuzzStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid fuzz: %w", err)
			}
			fuzz = v
		}
		out := Trim(src, fuzz)
		return out, nil

	case "annotate":
		// annotate text [fontPath] size x y color
		if !(len(args) == 5 || len(args) == 6) {
			return nil, fmt.Errorf("annotate requires 5 args: text size x y color or 6 args: text fontPath size x y color")
		}
		var text, fontPath, sizeStr, colorStr string
		var x, y int
		var size float64
		if len(args) == 5 {
			text = args[0]
			sizeStr = args[1]
			// parse x y
			tmpX, err := strconv.Atoi(args[2])
			if err != nil {
				return nil, fmt.Errorf("invalid x: %w", err)
			}
			tmpY, err := strconv.Atoi(args[3])
			if err != nil {
				return nil, fmt.Errorf("invalid y: %w", err)
			}
			x = tmpX
			y = tmpY
			colorStr = args[4]
		} else {
			// 6 args
			text = args[0]
			fontPath = args[1]
			sizeStr = args[2]
			tmpX, err := strconv.Atoi(args[3])
			if err != nil {
				return nil, fmt.Errorf("invalid x: %w", err)
			}
			tmpY, err := strconv.Atoi(args[4])
			if err != nil {
				return nil, fmt.Errorf("invalid y: %w", err)
			}
			x = tmpX
			y = tmpY
			colorStr = args[5]
		}
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size: %w", err)
		}
		col, err := parseHexColor(colorStr)
		if err != nil {
			return nil, fmt.Errorf("invalid color: %w", err)
		}
		out, err := Annotate(src, text, fontPath, size, x, y, col)
		return out, err

	case "composite":
		// composite srcImagePath composeOperator x y
		if len(args) != 4 {
			return nil, fmt.Errorf("composite requires 4 args: srcImagePath operator x y")
		}
		srcPath := args[0]
		op := args[1]
		xOff, err := strconv.Atoi(args[2])
		if err != nil {
			return nil, fmt.Errorf("invalid x: %w", err)
		}
		yOff, err := strconv.Atoi(args[3])
		if err != nil {
			return nil, fmt.Errorf("invalid y: %w", err)
		}
		f, err := os.Open(srcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open composite source: %w", err)
		}
		defer f.Close()
		img2, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("failed to decode composite source: %w", err)
		}
		out := Composite(src, img2, op, xOff, yOff)
		return out, nil

	case "identify":
		return nil, nil

	case "strip":
		// No-op for stdlib: re-encoding will drop metadata at save time
		return src, nil

	default:
		return nil, fmt.Errorf("unsupported command in stdlib engine: %s", commandName)
	}
}
