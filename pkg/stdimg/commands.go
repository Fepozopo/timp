// Package stdimg: authoritative registry of stdlib engine commands.
//
// This file mirrors the commands implemented in ApplyCommandStdlib in
// pkg/stdimg/engine.go. Keep this list up-to-date when you add or
// modify commands so callers (CLI, docs, help text) can read a single
// source of truth.

package stdimg

// ArgSpec describes a single argument for a command. Fields are textual
// and intended for help/validation UI rather than machine-enforced typing.
type ArgSpec struct {
	Name        string // human name
	Type        string // "int", "float", "bool", "string", "path", etc.
	Required    bool
	Default     string // textual default (for help only)
	Description string
}

// CommandSpec defines a single command and its expected arguments.
type CommandSpec struct {
	Name        string
	Args        []ArgSpec
	Usage       string // short usage string
	Description string // brief description
}

// Commands is the authoritative list of commands implemented by the stdlib engine.
// Keep this synchronized with ApplyCommandStdlib in pkg/stdimg/engine.go.
var Commands = []CommandSpec{
	{
		Name:        "resize",
		Args:        []ArgSpec{{"width", "int", true, "", "output width"}, {"height", "int", true, "", "output height"}},
		Usage:       "resize <width> <height>",
		Description: "Resize image using Lanczos resampling (a=3).",
	},
	{
		Name:        "rotate",
		Args:        []ArgSpec{{"degrees", "float", true, "", "rotation degrees"}},
		Usage:       "rotate <degrees>",
		Description: "Rotate image using inverse mapping with bilinear sampling.",
	},
	{
		Name:        "blur",
		Args:        []ArgSpec{{"sigma", "float", true, "", "gaussian sigma"}},
		Usage:       "blur <sigma>",
		Description: "Separable Gaussian blur.",
	},
	{
		Name:        "medianFilter",
		Args:        []ArgSpec{{"radius", "int", true, "", "median radius"}},
		Usage:       "medianFilter <radius>",
		Description: "Median filter (sliding-window histogram).",
	},
	{
		Name:        "despeckle",
		Args:        []ArgSpec{{"radius", "int", false, "1", "optional radius"}},
		Usage:       "despeckle [radius]",
		Description: "Despeckle (wrapper around median filter).",
	},
	{
		Name:        "level",
		Args:        []ArgSpec{{"blackPoint", "float", true, "", "black point"}, {"gamma", "float", true, "", "gamma"}, {"whitePoint", "float", true, "", "white point"}},
		Usage:       "level <blackPoint> <gamma> <whitePoint>",
		Description: "Adjust levels (black/gamma/white).",
	},
	{
		Name:        "normalize",
		Args:        []ArgSpec{},
		Usage:       "normalize",
		Description: "Stretch per-channel extremes to full [0,255].",
	},
	{
		Name:        "autoLevel",
		Args:        []ArgSpec{},
		Usage:       "autoLevel",
		Description: "Automatic level normalization.",
	},
	{
		Name:        "autoGamma",
		Args:        []ArgSpec{},
		Usage:       "autoGamma",
		Description: "Automatic gamma correction.",
	},
	{
		Name:        "gamma",
		Args:        []ArgSpec{{"gamma", "float", true, "", "gamma value"}},
		Usage:       "gamma <gamma>",
		Description: "Apply gamma correction.",
	},
	{
		Name:        "negate",
		Args:        []ArgSpec{{"onlyGray", "bool", false, "false", "only invert grayscale pixels"}},
		Usage:       "negate [onlyGray]",
		Description: "Invert colors (optional only-gray).",
	},
	{
		Name:        "threshold",
		Args:        []ArgSpec{{"value", "float", true, "", "threshold value"}, {"perChannel", "bool", false, "false", "apply per-channel"}},
		Usage:       "threshold <value> [perChannel]",
		Description: "Threshold image by value (luminance or per-channel).",
	},
	{
		Name:        "modulate",
		Args:        []ArgSpec{{"brightness", "float", true, "", "brightness percent (e.g., 100)"}, {"saturation", "float", true, "", "saturation percent"}, {"hue", "float", true, "", "hue degrees"}},
		Usage:       "modulate <brightness> <saturation> <hue>",
		Description: "Adjust brightness, saturation and hue.",
	},
	{
		Name:        "vignette",
		Args:        []ArgSpec{{"radius", "float", true, "", "radius"}, {"sigma", "float", true, "", "sigma"}, {"x", "int", true, "", "center x"}, {"y", "int", true, "", "center y"}, {"strength", "float", false, "1.0", "0..1 or percent like 50%"}},
		Usage:       "vignette <radius> <sigma> <x> <y> [strength]",
		Description: "Apply vignette effect centered at (x,y).",
	},
	{
		Name:        "grayscale",
		Args:        []ArgSpec{},
		Usage:       "grayscale",
		Description: "Convert to luminance (Rec.709).",
	},
	{
		Name:        "edge",
		Args:        []ArgSpec{{"sigma", "float", false, "0.0", "pre-blur sigma"}, {"scale", "float", false, "1.0", "edge scale multiplier"}, {"threshold", "float", false, "0.0", "threshold value"}, {"binary", "bool", false, "false", "binary output"}},
		Usage:       "edge [sigma] [scale] [threshold] [binary]",
		Description: "Sobel-based edge detector with options.",
	},
	{
		Name:        "adaptiveBlur",
		Args:        []ArgSpec{{"radius", "float", false, "1.0", "variance neighborhood radius"}, {"sigmaMin", "float", false, "0.5", "min sigma (for high variance)"}, {"sigmaMax", "float", false, "1.0", "max sigma (for low variance)"}, {"levels", "int", false, "6", "discrete levels to precompute"}},
		Usage:       "adaptiveBlur [radius] [sigmaMin] [sigmaMax] [levels]",
		Description: "Variance-driven per-pixel adaptive blur.",
	},
	// New adaptive/stdlib commands added during migration
	{
		Name:        "adaptiveResize",
		Args:        []ArgSpec{{"width", "int", false, "0", "target width (0 = preserve aspect)"}, {"height", "int", false, "0", "target height (0 = preserve aspect)"}, {"a", "float", false, "3.0", "Lanczos a parameter (3 recommended)"}},
		Usage:       "adaptiveResize [width] [height] [a]",
		Description: "Resize using Lanczos resampling with aspect-preserve semantics.",
	},
	{
		Name:        "adaptiveSharpen",
		Args:        []ArgSpec{{"radius", "float", false, "0.0", "blur radius (0 = auto)"}, {"sigma", "float", false, "1.0", "sigma for blur"}, {"amount", "float", false, "1.0", "sharpen amount"}},
		Usage:       "adaptiveSharpen [radius] [sigma] [amount]",
		Description: "Sharpen using unsharp-mask (approximation).",
	},
	{
		Name:        "adaptiveThreshold",
		Args:        []ArgSpec{{"window_width", "int", false, "15", "local window width (odd)"}, {"window_height", "int", false, "15", "local window height (odd)"}, {"offset", "float", false, "0.0", "threshold offset (subtract from mean)"}},
		Usage:       "adaptiveThreshold [window_width] [window_height] [offset]",
		Description: "Local threshold using moving window mean (bilevel output).",
	},
	{
		Name:        "addNoise",
		Args:        []ArgSpec{{"type", "enum", false, "GAUSSIAN", "noise type (GAUSSIAN|UNIFORM|POISSON)"}, {"amount", "float", false, "10.0", "noise strength (stddev or range)"}, {"seed", "int", false, "0", "random seed (0 = time-based)"}},
		Usage:       "addNoise [type] [amount] [seed]",
		Description: "Add noise to image; supports GAUSSIAN, UNIFORM (POISSON optional).",
	},
	{
		Name:        "crop",
		Args:        []ArgSpec{{"width", "int", true, "", "crop width"}, {"height", "int", true, "", "crop height"}, {"x", "int", true, "", "x offset"}, {"y", "int", true, "", "y offset"}},
		Usage:       "crop <width> <height> <x> <y>",
		Description: "Crop image (intersected with bounds).",
	},
	{
		Name:        "flip",
		Args:        []ArgSpec{},
		Usage:       "flip",
		Description: "Vertical flip.",
	},
	{
		Name:        "flop",
		Args:        []ArgSpec{},
		Usage:       "flop",
		Description: "Horizontal flip.",
	},
	{
		Name: "histogram",
		Args: []ArgSpec{
			{"bins", "int", false, "256", "number of bins"},
			{"pixelWindow", "int", false, "20", "smoothing window in pixels (default 20). Decrease to zoom out (show more narrow spikes); increase to zoom in (more smoothing)."},
		},
		Usage:       "histogram [bins] [pixelWindow]",
		Description: "Render a histogram image (returns image)",
	},
	{
		Name:        "equalize",
		Args:        []ArgSpec{},
		Usage:       "equalize",
		Description: "Equalize histogram per-channel.",
	},
	{
		Name:        "trim",
		Args:        []ArgSpec{{"fuzz", "float_or_percent", true, "", "fuzz numeric or percent (e.g. 5 or 5%)"}},
		Usage:       "trim <fuzz>",
		Description: "Trim borders within fuzz tolerance.",
	},
	{
		Name:        "floodfillPaint",
		Args:        []ArgSpec{{"fillColor", "string", true, "", "CSS color or hex (e.g. #ff0000)"}, {"fuzz", "float_or_percent", true, "", "fuzz as Lab delta-E or percent (e.g. 5 or 50%)"}, {"borderColor", "string", false, "", "CSS color or hex for border or empty string"}, {"x", "int", true, "", "start x"}, {"y", "int", true, "", "start y"}, {"invert", "bool", false, "false", "invert fill region"}},
		Usage:       "floodfillPaint <fillColor> <fuzz> <borderColor> <x> <y> [invert]",
		Description: "Flood-fill region starting at (x,y) using perceptual fuzz (Lab delta-E).",
	},
	{
		Name:        "annotate",
		Args:        []ArgSpec{{"text", "string", true, "", "text to draw"}, {"fontPath", "path_or_empty", false, "", "font path (optional)"}, {"size", "float", true, "", "font size"}, {"x", "int", true, "", "x position"}, {"y", "int", true, "", "y position"}, {"color", "string", true, "", "CSS hex or name (e.g. #ff0000)"}},
		Usage:       "annotate <text> [fontPath] <size> <x> <y> <color>",
		Description: "Draw text; supports 5 or 6 args (font optional).",
	},
	{
		Name:        "composite",
		Args:        []ArgSpec{{"srcImagePath", "path", true, "", "path to source image"}, {"operator", "string", true, "", "compose operator (e.g. OVER)"}, {"x", "int", true, "", "x offset"}, {"y", "int", true, "", "y offset"}},
		Usage:       "composite <srcImagePath> <operator> <x> <y>",
		Description: "Composite an image loaded from disk at offset using operator.",
	},
	{
		Name:        "identify",
		Args:        []ArgSpec{},
		Usage:       "identify",
		Description: "Print image metadata; returns nil image.",
	},
	{
		Name:        "strip",
		Args:        []ArgSpec{},
		Usage:       "strip",
		Description: "Strip metadata; returns image unchanged.",
	},
	{
		Name:        "sepia",
		Args:        []ArgSpec{{"percentage", "float_or_percent", false, "70%", "sepia intensity (0..100% or 0..1)"}, {"midtoneCenter", "float", false, "50", "midtone center L (0..100)"}, {"midtoneSigma", "float", false, "20", "midtone width (sigma)"}, {"highlightThreshold", "float", false, "80", "L at which protection starts"}, {"highlightSoftness", "float", false, "10", "softness for highlight protection"}, {"curve", "float", false, "0.12", "filmic S-curve strength (0..1)"}},
		Usage:       "sepia [percentage] [midtoneCenter] [midtoneSigma] [highlightThreshold] [highlightSoftness] [curve]",
		Description: "Apply Sepia tone with optional intensity and tonal controls.",
	},
}
