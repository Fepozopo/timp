package cli

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

// TestPreviewInlineSequence verifies that PreviewImage emits an inline-image OSC
// sequence when TERM_PROGRAM indicates an inline-capable terminal.
func TestPreviewInlineSequence(t *testing.T) {
	// Make a tiny test image
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(1, 1, color.RGBA{255, 255, 0, 255})

	// Force inline-capable detection and ensure we don't hit kitty heuristics
	os.Setenv("TERM_PROGRAM", "WezTerm")
	oldTerm := os.Getenv("TERM")
	os.Setenv("TERM", "xterm-256color")
	defer func() {
		os.Unsetenv("TERM_PROGRAM")
		if oldTerm == "" {
			os.Unsetenv("TERM")
		} else {
			os.Setenv("TERM", oldTerm)
		}
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	os.Stdout = w

	// Call preview
	if err := PreviewImage(img, "png"); err != nil {
		t.Fatalf("PreviewImage error: %v", err)
	}

	// Close writer and restore stdout
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	out := buf.String()
	if !strings.Contains(out, "1337") && !strings.Contains(out, "\x1b]1337") {
		t.Fatalf("expected inline 1337 sequence in output, got: %q", out)
	}
}

// TestPreviewEncodesJPEG ensures that when format=="jpeg" PreviewImage encodes JPEG bytes
// (i.e., the embedded base64 payload begins with JPEG SOI 0xFF 0xD8).
func TestPreviewEncodesJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{10, 20, 30, 255})

	os.Setenv("TERM_PROGRAM", "WezTerm")
	oldTerm := os.Getenv("TERM")
	os.Setenv("TERM", "xterm-256color")
	defer func() {
		os.Unsetenv("TERM_PROGRAM")
		if oldTerm == "" {
			os.Unsetenv("TERM")
		} else {
			os.Setenv("TERM", oldTerm)
		}
	}()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	os.Stdout = w

	if err := PreviewImage(img, "jpeg"); err != nil {
		t.Fatalf("PreviewImage error: %v", err)
	}
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout
	out := buf.String()

	// find inline base64 payload after ':' and before BEL or ESC
	idx := strings.Index(out, ":")
	if idx < 0 {
		t.Fatalf("no ':' found in output: %q", out)
	}
	payload := out[idx+1:]
	// cut at BEL or ESC
	if bi := strings.Index(payload, "\a"); bi >= 0 {
		payload = payload[:bi]
	}
	if bi := strings.Index(payload, "\x1b"); bi >= 0 {
		payload = payload[:bi]
	}
	// decode base64
	dec, derr := base64.StdEncoding.DecodeString(payload)
	if derr != nil {
		t.Fatalf("base64 decode failed: %v", derr)
	}
	if len(dec) < 2 || dec[0] != 0xFF || dec[1] != 0xD8 {
		t.Fatalf("expected JPEG SOI bytes, got: %x", dec[:4])
	}
}
