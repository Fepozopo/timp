package cli

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Terminal preview helper for Kitty and iTerm2 inline-image protocols.
//
// Behavior:
//   - If kitty is detected (KITTY_WINDOW_ID or TERM contains "kitty"), the PNG is sent using
//     the kitty graphics protocol (chunked base64 inside ESC _G ... ESC \).
//   - Else if iTerm2 is detected (TERM_PROGRAM == "iTerm.app" || ITERM_SESSION_ID present),
//     the PNG is sent using the iTerm2 OSC 1337 inline file sequence.
//   - Else if other terminals known to support inline images (WezTerm, Warp, Tabby, VSCode, etc)
//     the same iTerm2-style OSC 1337 sequence is used.
//   - Else if a terminal likely to support Sixel graphics is detected (foot, Windows Terminal, st with sixel patch, etc),
//     the PNG is piped to an external sixel renderer (img2sixel or chafa).
//   - Else, if chafa is available on PATH, it will be invoked to render a terminal-friendly approximation
//     even for terminals that don't implement the above protocols.
//   - If none is available, returns an error indicating no supported terminal.
//
// Notes:
//   - Sending binary escape sequences to stdout is expected in this terminal-only preview mode.
//
// Debugging helper controlled by PREVIEW_DEBUG=1
var previewDebug bool

func init() {
	err := godotenv.Load()
	if err != nil {
		// Ignore error if .env not present; it's optional
	}

	debug := os.Getenv("PREVIEW_DEBUG")
	if debug == "1" || debug == "true" {
		previewDebug = true
	}
}

func debugf(format string, args ...interface{}) {
	if previewDebug {
		fmt.Fprintf(os.Stderr, "timp-preview: "+format+"\n", args...)
	}
}

func isKitty() bool {
	// Primary hint that the terminal is kitty or a kitty-compatible implementation
	// (e.g. ghostty exposes the kitty compatibility features).
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	// Inspect TERM for known kitty-compatible names.
	term := strings.ToLower(os.Getenv("TERM"))
	// Accept kitty and ghostty (and short 'ghost') as kitty-compatible terminals.
	if strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") || strings.Contains(term, "ghost") {
		return true
	}
	// Konsole may implement parts of the protocol via an older kitty compatibility mode.
	if os.Getenv("KONSOLE_VERSION") != "" {
		return true
	}
	return false
}

// Detects terminals that implement the generic "inline images" OSC protocol
// (iTerm2 style) — many modern terminal emulators (WezTerm, Warp, Tabby, VSCode's terminal,
// Rio, Hyper, Bobcat and others) implement that or compatible behavior.
// We use a heuristic based on TERM_PROGRAM and common TERM substrings.
func isInlineImageCapable() bool {
	debugf("checking inline-image capability via TERM_PROGRAM/TERM")
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "Warp", "Hyper", "vscode", "VSCode", "Tabby", "Bobcat":
		debugf("TERM_PROGRAM indicates inline-capable: %s", os.Getenv("TERM_PROGRAM"))
		return true
	}
	// Some terminals expose recognizable TERM values
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "wezterm") || strings.Contains(term, "warp") || strings.Contains(term, "tabby") ||
		strings.Contains(term, "vscode") || strings.Contains(term, "wez") {
		debugf("TERM suggests inline-capable: %s", term)
		return true
	}
	// A direct iTerm2 hint
	if os.Getenv("ITERM_SESSION_ID") != "" || os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		debugf("iTerm2 indicators present")
		return true
	}
	return false
}

// Detect terminals that likely support Sixel graphics (foot, Windows Terminal >= certain versions,
// st with sixel patch, Black Box, etc). This is heuristic — if you rely on Sixel in CI, add
// a user-configurable override environment variable SIXEL_PREVIEW=1 to force it.
func isSixelCapable() bool {
	if os.Getenv("SIXEL_PREVIEW") == "1" {
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "foot") || strings.Contains(term, "st") || strings.Contains(term, "linux") {
		return true
	}
	if os.Getenv("WT_SESSION") != "" { // Windows Terminal newer versions support sixel
		return true
	}
	return false
}

// hasChafa reports whether the external 'chafa' binary is available in PATH.
// We treat chafa as a usable fallback for terminals that don't implement inline
// or sixel protocols but can still display block/character graphics.
func hasChafa() bool {
	if os.Getenv("CHAFAPREVIEW") == "1" {
		return true
	}
	if _, err := exec.LookPath("chafa"); err == nil {
		return true
	}
	return false
}

// postImageNewlines returns a sane number of newline lines to emit after an image
// is rendered. It uses hints like the requested rows (from kitty placement) or
// the chafa size if provided. The result is clamped to avoid emitting a large
// gap; default is 1-3 lines depending on image height hints.
func postImageNewlines(requestedRows int) int {
	// If explicit KITTY_PREVIEW_ROWS is set, prefer that.
	if v := os.Getenv("KITTY_PREVIEW_ROWS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			if n == 0 {
				return 1
			}
			if n <= 3 {
				return n
			}
			if n <= 10 {
				return 3
			}
			return 4
		}
	}

	// If chafa provided a requested row count, use it to pick 1-4 lines.
	if requestedRows > 0 {
		if requestedRows <= 2 {
			return 1
		}
		if requestedRows <= 6 {
			return 2
		}
		if requestedRows <= 20 {
			return 3
		}
		return 4
	}

	// Default small padding to ensure prompt shows below image.
	return 1
}

// PreviewSupported returns true if the running environment likely supports a terminal inline preview.
// We consider chafa availability as a valid fallback even if no inline/sixel protocol is detected.
func PreviewSupported() bool {
	supported := isKitty() || isInlineImageCapable() || isSixelCapable() || hasChafa()
	debugf("PreviewSupported -> %v (kitty=%v inline=%v sixel=%v chafa=%v)", supported, isKitty(), isInlineImageCapable(), isSixelCapable(), hasChafa())
	return supported
}

// PreviewImage encodes an image.Image to the requested container format and previews it in terminal.
// format should be a lowercase string like "png" or "jpeg". If empty or unrecognized, PNG is used.
func PreviewImage(img image.Image, format string) error {
	if img == nil {
		return fmt.Errorf("nil image")
	}
	var buf bytes.Buffer
	f := strings.ToLower(format)
	if f == "jpeg" || f == "jpg" {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
			return fmt.Errorf("jpeg encode failed: %w", err)
		}
	} else {
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("png encode failed: %w", err)
		}
		f = "png"
	}
	return previewBytes(buf.Bytes(), f)
}

// previewBytes centralizes the logic of sending bytes via kitty/inline/sixel/chafa.
func previewBytes(blob []byte, format string) error {
	if len(blob) == 0 {
		return fmt.Errorf("empty image blob")
	}

	// Prefer kitty if available (unicode placeholders / placement)
	if isKitty() {
		debugf("attempting kitty protocol")
		if err := sendKittyImage(blob, format); err != nil {
			debugf("kitty protocol failed: %v", err)
			// try other fallbacks
			if isInlineImageCapable() {
				if err2 := sendInlineImage(blob, format); err2 == nil {
					return nil
				}
			}
			if isSixelCapable() {
				if err3 := sendSixelImage(blob, format); err3 == nil {
					return nil
				}
			}
			if hasChafa() {
				if err4 := sendChafaImage(blob, format); err4 == nil {
					return nil
				}
			}
			return fmt.Errorf("kitty preview failed: %w", err)
		}
		return nil
	}

	if isInlineImageCapable() {
		if err := sendInlineImage(blob, format); err != nil {
			if isSixelCapable() {
				if err2 := sendSixelImage(blob, format); err2 == nil {
					return nil
				}
			}
			if hasChafa() {
				if err3 := sendChafaImage(blob, format); err3 == nil {
					return nil
				}
			}
			return fmt.Errorf("inline image preview failed: %w", err)
		}
		return nil
	}

	if isSixelCapable() {
		if err := sendSixelImage(blob, format); err != nil {
			if hasChafa() {
				if err2 := sendChafaImage(blob, format); err2 == nil {
					return nil
				}
			}
			return fmt.Errorf("sixel preview failed: %w", err)
		}
		return nil
	}

	if hasChafa() {
		if err := sendChafaImage(blob, format); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no preview protocol matched")
}

// sendKittyImage sends encoded image bytes to the terminal using the kitty graphics protocol.
// It chunks base64 payload into <=4096-byte chunks per spec. The first chunk includes
// placement parameters to force the image to render into a fixed area (columns x rows).
//
// The function accepts raw image bytes in `data` and a `format` hint (e.g. "png" or "jpeg").
// Placement sizing is controlled by environment variables (optional):
//
//	KITTY_PREVIEW_COLS and KITTY_PREVIEW_ROWS
//
// If those are not present, sensible defaults are used.
//
// Note: when sending PNG the implementation uses f=100; for JPEG it may include a numeric f= hint.
// We suppress terminal responses with q=2.
func sendKittyImage(data []byte, format string) error {
	if len(data) == 0 {
		return fmt.Errorf("no data")
	}

	debugf("sendKittyImage preparing to send %d bytes (raw %s)", len(data), format)

	enc := base64.StdEncoding.EncodeToString(data)
	const chunkSize = 4096

	// Determine preview placement size from environment (defaults).
	cols := 60
	rows := 20
	if v := os.Getenv("KITTY_PREVIEW_COLS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cols = n
		}
	}
	if v := os.Getenv("KITTY_PREVIEW_ROWS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rows = n
		}
	}

	debugf("kitty placement: cols=%d rows=%d (requested)", cols, rows)

	stdout := os.Stdout

	// Helper to write a raw sequence to stdout.
	writeSeq := func(s string) error {
		_, err := stdout.Write([]byte(s))
		return err
	}

	total := len(enc)
	first := true
	for pos := 0; pos < total; pos += chunkSize {
		end := pos + chunkSize
		if end > total {
			end = total
		}
		chunk := enc[pos:end]
		last := end == total

		mVal := "0"
		if !last {
			mVal = "1"
		}

		if first {
			// First chunk includes full control keys and placement (c,r).
			// a=T transmit+display, t=d direct payload, q=2 suppress responses,
			// c=<cols>, r=<rows> request rendering area.
			// Include an explicit `f=` token for PNG to match kitty expectations.
			fTok := ""
			if strings.HasPrefix(strings.ToLower(format), "png") {
				fTok = "f=100,"
			} else if strings.HasPrefix(strings.ToLower(format), "j") {
				// common kitty numeric id for JPEG is 24 (used by some implementations);
				// include it as a hint — if incorrect terminals should still try to detect.
				fTok = "f=24,"
			}
			header := fmt.Sprintf("\x1b_Ga=T,%st=d,q=2,c=%d,r=%d,m=%s;", fTok, cols, rows, mVal)
			header += chunk + "\x1b\\"
			if err := writeSeq(header); err != nil {
				return err
			}
			first = false
			continue
		}

		// Subsequent chunks must contain only m=1/m=0 and the payload chunk.
		header := "\x1b_G" + "m=" + mVal + ";" + chunk + "\x1b\\"
		if err := writeSeq(header); err != nil {
			return err
		}
	}

	// After the image is transmitted, advance the cursor a small number of lines
	// so subsequent text appears directly under the image. Use environment
	// hints (KITTY_PREVIEW_ROWS / CHAFA_SIZE) when available and clamp to a
	// small maximum to avoid a large gap.
	for i := 0; i < postImageNewlines(rows); i++ {
		fmt.Println()
	}

	// Done
	return nil
}

// sendInlineImagePNG emits the generic iTerm2-style inline image OSC (1337) sequence.
// Many terminals implement a compatible inline-image OSC (iTerm2, WezTerm, Warp, Tabby, VSCode, etc).
// Format: ESC ] 1337 ; File=inline=1;size=<n> : <base64> BEL
func sendInlineImagePNG(data []byte) error {
	return sendInlineImage(data, "png")
}

// sendInlineImage emits the generic iTerm2-style inline image OSC (1337) sequence
// using a name hint derived from format.
func sendInlineImage(data []byte, format string) error {
	if len(data) == 0 {
		return fmt.Errorf("no data")
	}
	debugf("sendInlineImage preparing to send %d bytes (format=%s)", len(data), format)
	enc := base64.StdEncoding.EncodeToString(data)
	name := "preview.png"
	if strings.HasPrefix(strings.ToLower(format), "j") {
		name = "preview.jpg"
	}
	seq := "\x1b]1337;File=name=" + name + ";inline=1;size=" + fmt.Sprintf("%d", len(data)) + ":" + enc + "\a"
	n, err := os.Stdout.Write([]byte(seq))
	debugf("wrote %d bytes to stdout for inline image (err=%v)", n, err)

	// After the image is transmitted, advance the cursor a small number of lines
	// so the prompt/info prints directly under the image instead of far below.
	for i := 0; i < postImageNewlines(0); i++ {
		fmt.Println()
	}

	return err
}

// sendSixelImage attempts to render image data using an external sixel renderer (img2sixel).
// It pipes the provided image bytes (`data`) to the external tool which is expected to emit sixel to stdout.
// The `format` parameter is a hint (e.g. "png" or "jpeg") and may influence fallbacks.
// This is a pragmatic approach because implementing a sixel encoder here is beyond scope.
func sendSixelImage(data []byte, format string) error {
	if len(data) == 0 {
		return fmt.Errorf("no data")
	}

	debugf("sendSixelImage attempting img2sixel (or chafa) for %d bytes (format=%s)", len(data), format)

	// Try to locate a suitable external sixel tool.
	// Common tool: img2sixel (part of libsixel or some distributions).
	// We call it with '-' to accept stdin.
	cmd := exec.Command("img2sixel", "-")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err == nil {
		debugf("img2sixel succeeded")
		// Advance a small number of lines after the image so subsequent text
		// appears just below it.
		for i := 0; i < postImageNewlines(0); i++ {
			fmt.Println()
		}
		return nil
	} else {
		debugf("img2sixel failed: %v", err)
	}

	// If img2sixel isn't available, try chafa as a fallback (chafa supports multiple terminals).
	if err := sendChafaImage(data, format); err == nil {
		debugf("chafa succeeded")
		// sendChafaPNG already advances the cursor; don't print extra lines here.
		return nil
	} else {
		debugf("chafa failed: %v", err)
	}

	// As a last resort, write a small inline PNG with base64 to the terminal (rarely supported).
	debugf("falling back to inline PNG base64 sequence as last resort")
	// Last-resort inline hint with name reflecting format.
	enc := base64.StdEncoding.EncodeToString(data)
	name := "preview.png"
	if strings.HasPrefix(strings.ToLower(format), "j") {
		name = "preview.jpg"
	}
	seq := "\x1b]1337;File=name=" + name + ";inline=1;size=" + fmt.Sprintf("%d", len(data)) + ":" + enc + "\a"
	n, err := os.Stdout.Write([]byte(seq))
	debugf("wrote %d bytes for inline PNG fallback (err=%v)", n, err)

	// Ensure the cursor moves to the next line after the image.
	for i := 0; i < postImageNewlines(0); i++ {
		fmt.Println()
	}

	return err
}

// sendChafaImage invokes chafa to render the provided image bytes to stdout.
// It attempts to choose reasonable flags to produce a block-symbol rendering that
// works in many terminals. The function accepts `data` and a `format` hint (e.g. "png").
// It returns an error if chafa is not present or fails.
func sendChafaImage(data []byte, format string) error {
	if len(data) == 0 {
		return fmt.Errorf("no data")
	}

	// Allow an environment override to skip attempting chafa when explicitly disabled.
	if os.Getenv("NO_CHAFA") == "1" {
		return fmt.Errorf("chafa usage disabled via NO_CHAFA=1")
	}

	// Ensure chafa exists
	if _, err := exec.LookPath("chafa"); err != nil {
		return fmt.Errorf("chafa not found in PATH: %w", err)
	}

	debugf("sendChafaImage invoking chafa for %d bytes (format=%s)", len(data), format)

	// Determine chafa args. Use block fill and symbols for dense output.
	// Default size is 80x40; user can override via CHAFA_SIZE.
	args := []string{"--fill=block", "--symbols=block", "-s", "80x40", "-"}

	if v := os.Getenv("CHAFA_SIZE"); v != "" {
		// If the user provides a size override, pass it through to -s.
		args = []string{"--fill=block", "--symbols=block", "-s", v, "-"}
	}

	// Allow custom fill/symbol selection via env (optional)
	if f := os.Getenv("CHAFA_FILL"); f != "" {
		// replace --fill value
		for i, a := range args {
			if strings.HasPrefix(a, "--fill=") {
				args[i] = "--fill=" + f
			}
		}
	}
	if s := os.Getenv("CHAFA_SYMBOLS"); s != "" {
		for i, a := range args {
			if strings.HasPrefix(a, "--symbols=") {
				args[i] = "--symbols=" + s
			}
		}
	}

	cmd := exec.Command("chafa", args...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chafa failed: %w", err)
	}

	// Ensure adequate spacing after the image so subsequent text isn't overwritten.
	// Use CHAFA_SIZE hint if provided; otherwise advance only a small number
	// of lines so the prompt prints just under the rendered output.
	sizeRows := 0
	if v := os.Getenv("CHAFA_SIZE"); v != "" {
		parts := strings.Split(v, "x")
		if len(parts) == 2 {
			if h, err := strconv.Atoi(parts[1]); err == nil {
				sizeRows = h
			}
		}
	}
	for i := 0; i < postImageNewlines(sizeRows); i++ {
		fmt.Println()
	}

	return nil
}
