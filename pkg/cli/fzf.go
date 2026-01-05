package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Fepozopo/timp/pkg/stdimg"
)

// SelectCommandWithFzfStd displays a list of stdimg commands in fzf and returns the selected command name.
func SelectCommandWithFzfStd(commands []stdimg.CommandSpec) (string, error) {
	var b strings.Builder
	for _, c := range commands {
		// format as "name: description"
		b.WriteString(fmt.Sprintf("%s: %s\n", c.Name, c.Description))
	}

	cmd := exec.Command("fzf")
	cmd.Stdin = strings.NewReader(b.String())

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running fzf: %w", err)
	}

	selection := strings.TrimSpace(out.String())
	parts := strings.SplitN(selection, ":", 2)
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0]), nil
	}

	return "", fmt.Errorf("no command selected")
}

// SelectFileWithFzf launches fzf with a list of common image files found under startDir.
// It returns the full path of the selected file or an error if selection failed.
//
// This implementation reuses the terminal detection helpers in terminal_preview.go
// (isKitty, isInlineImageCapable, isSixelCapable, PreviewSupported) to choose a
// reasonable --preview command for fzf. The preview will attempt to use the most
// capable renderer available for the detected terminal.
//
// Note: This implementation shells out to `find` piped into `fzf`. It requires both
// `find` and `fzf` to be available in PATH. startDir may be "." or any directory path.
func SelectFileWithFzf(startDir string) (string, error) {
	// Quote the directory to safely handle spaces/special chars.
	quotedDir := strconv.Quote(startDir)

	// Build a terminal-aware preview command for fzf. The preview command uses
	// fzf's {} replacement for the current file path. We prefer inline/kitty/sixel
	// renderers when the terminal detection indicates support; otherwise fall back
	// to `chafa` for pixelated rendering or textual preview.
	//
	// The preview command tries multiple renderers in order, using `||` to fall
	// back if the preferred renderer is not available. Errors are redirected to
	// /dev/null to avoid cluttering the preview pane.
	//
	// Note: fzf's --preview option does not support complex shell constructs like
	// conditionals or functions, so we must use a single command line with `||`
	// chains to achieve fallback behavior.
	//
	// We also include a control sequence to clear kitty images before rendering
	// a new image, to avoid accumulating images in the terminal buffer.
	var previewCmd string

	// Helper chains: try best renderer, then fall back to others or textual viewers.
	if isKitty() {
		// Prefer kitty icat. If unavailable, try chafa.
		previewCmd = "printf \"\\x1b_Ga=d\\x1b\\\\\"; kitty +kitten icat --silent {} 2>/dev/null || chafa --fill=block --symbols=block -s 80x40 {} 2>/dev/null"
	} else if isInlineImageCapable() {
		// Prefer imgcat (iTerm2 integration). If not present, try chafa.
		previewCmd = "imgcat {} 2>/dev/null  || chafa --fill=block --symbols=block -s 80x40 {} 2>/dev/null"
	} else if isSixelCapable() {
		// Prefer sixel renderers. If img2sixel not present, try chafa.
		previewCmd = "img2sixel {} 2>/dev/null || chafa --fill=block --symbols=block -s 80x40 {} 2>/dev/null"
	} else {
		// No detected image-capable terminal: use pixel renderer if present, else textual preview.
		previewCmd = "chafa --fill=block --symbols=block -s 80x40 {} 2>/dev/null"
	}

	// Build the find + fzf command. Escape percent signs in the format string.
	// Use --preview-window to allocate space on the right for the preview.
	cmdStr := fmt.Sprintf(
		"find %s -type f \\( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' -o -iname '*.gif' -o -iname '*.tif' -o -iname '*.tiff' \\) | fzf --height 100%% --border --prompt='Files> ' --ansi --preview=%q --preview-window='right:60%%'",
		quotedDir,
		previewCmd,
	)
	cmd := exec.Command("bash", "-lc", cmdStr)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// attempt to clear kitty images regardless of error
		clearKittyImages()
		return "", fmt.Errorf("error running fzf for files: %w", err)
	}

	// clear preview images left behind by the previewer (kitty graphics)
	clearKittyImages()

	selection := strings.TrimSpace(out.String())
	if selection == "" {
		return "", fmt.Errorf("no file selected")
	}
	return selection, nil
}

// clearKittyImages emits the kitty graphics "delete" control sequence.
// Terminals that don't understand it will ignore it.
func clearKittyImages() {
	// ESC _ G a=d ESC \
	// We write to stdout so the control sequence targets the foreground terminal.
	fmt.Fprint(os.Stdout, "\x1b_Ga=d\x1b\\")
}
