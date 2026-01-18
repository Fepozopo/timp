package cli

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

// makeExifPayload builds a minimal EXIF APP1 payload (starting with "Exif\x00\x00")
// containing a single Orientation tag (0x0112) in IFD0 with the provided value.
func makeExifPayload(orientation uint16) []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte("Exif\x00\x00"))
	// TIFF header: little-endian 'II', magic 0x2A, offset to IFD0 = 8
	buf.Write([]byte{'I', 'I'})
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x2A))
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	// IFD0: 1 entry
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	// Entry: tag 0x0112 (Orientation), type SHORT (3), count 1
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0112))
	_ = binary.Write(buf, binary.LittleEndian, uint16(3))
	_ = binary.Write(buf, binary.LittleEndian, uint32(1))
	// Value (4 bytes) - SHORT value placed in first two bytes
	_ = binary.Write(buf, binary.LittleEndian, uint16(orientation))
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))
	// next IFD offset = 0
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	return buf.Bytes()
}

func makeTestJPEGWithSegments(t *testing.T, exifOrientation uint16) ([]byte, []AppSegment) {
	// create a small image
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	// fill with a color
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 10), uint8(y * 10), 128, 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("jpeg encode failed: %v", err)
	}
	jpegBytes := buf.Bytes()
	// craft APPn segments
	exifPayload := makeExifPayload(exifOrientation)
	segs := []AppSegment{
		{Marker: 0xE0, Payload: []byte("JFIF\x00dummy")},
		{Marker: 0xE1, Payload: exifPayload},
		{Marker: 0xE2, Payload: []byte("XMPDATA")},
	}
	final, err := insertAppSegmentsIntoJPEG(jpegBytes, segs)
	if err != nil {
		t.Fatalf("insertAppSegmentsIntoJPEG failed: %v", err)
	}
	return final, segs
}

func TestAppSegmentsRoundTrip(t *testing.T) {
	origBytes, origSegs := makeTestJPEGWithSegments(t, 6)
	f, err := os.CreateTemp("", "orig-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write(origBytes)
	f.Close()

	// LoadImage should parse the APPn segments and auto-orient
	img, _, parsedSegs, autoOriented, err := LoadImage(f.Name())
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}
	if img == nil {
		t.Fatalf("expected image, got nil")
	}
	if !autoOriented {
		t.Fatalf("expected autoOriented true for orientation 6")
	}
	if len(parsedSegs) != len(origSegs) {
		t.Fatalf("expected %d parsed segments, got %d", len(origSegs), len(parsedSegs))
	}
	// non-EXIF payloads should match exactly
	for i := range origSegs {
		if origSegs[i].Marker == 0xE1 {
			continue
		}
		if parsedSegs[i].Marker != origSegs[i].Marker {
			t.Fatalf("marker mismatch at %d: want 0x%02X got 0x%02X", i, origSegs[i].Marker, parsedSegs[i].Marker)
		}
		if !bytes.Equal(parsedSegs[i].Payload, origSegs[i].Payload) {
			t.Fatalf("payload mismatch at %d", i)
		}
	}

	// SaveImage should reinsert segments; EXIF orientation should be set to 1
	outF, err := os.CreateTemp("", "out-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	outF.Close()
	defer os.Remove(outF.Name())
	if err := SaveImage(outF.Name(), img, parsedSegs, autoOriented); err != nil {
		t.Fatalf("SaveImage failed: %v", err)
	}
	outBytes, err := os.ReadFile(outF.Name())
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	reParsed, err := parseJPEGAppSegments(outBytes)
	if err != nil {
		t.Fatalf("parseJPEGAppSegments failed: %v", err)
	}
	if len(reParsed) != len(origSegs) {
		t.Fatalf("expected %d re-parsed segments, got %d", len(origSegs), len(reParsed))
	}
	// APP1 payload should now have orientation 1
	orient, oerr := extractJPEGOrientation(outBytes)
	if oerr != nil {
		t.Fatalf("extractJPEGOrientation failed: %v", oerr)
	}
	if orient != 1 {
		t.Fatalf("expected orientation 1 after save, got %d", orient)
	}
}

func TestStripRemovesAppSegments(t *testing.T) {
	origBytes, _ := makeTestJPEGWithSegments(t, 1)
	f, err := os.CreateTemp("", "orig2-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write(origBytes)
	f.Close()

	img, _, parsedSegs, _, err := LoadImage(f.Name())
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}
	// simulate strip by passing nil segments
	outF, err := os.CreateTemp("", "out2-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	outF.Close()
	defer os.Remove(outF.Name())
	if err := SaveImage(outF.Name(), img, nil, false); err != nil {
		t.Fatalf("SaveImage failed: %v", err)
	}
	outBytes, err := os.ReadFile(outF.Name())
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	reParsed, err := parseJPEGAppSegments(outBytes)
	if err != nil {
		t.Fatalf("parseJPEGAppSegments failed: %v", err)
	}
	if len(reParsed) != 0 {
		t.Fatalf("expected 0 app segments after strip, got %d", len(reParsed))
	}
	_ = parsedSegs // keep var referenced in case future assertions are added
}

func TestAutoOrientSetsExifOrientationOne(t *testing.T) {
	origBytes, origSegs := makeTestJPEGWithSegments(t, 3)
	f, err := os.CreateTemp("", "orig3-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write(origBytes)
	f.Close()

	img, _, parsedSegs, autoOriented, err := LoadImage(f.Name())
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}
	if !autoOriented {
		t.Fatalf("expected autoOriented true for orientation 3")
	}
	// Save and verify orientation in saved bytes
	outF, err := os.CreateTemp("", "out3-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	outF.Close()
	defer os.Remove(outF.Name())
	if err := SaveImage(outF.Name(), img, parsedSegs, autoOriented); err != nil {
		t.Fatalf("SaveImage failed: %v", err)
	}
	outBytes, err := os.ReadFile(outF.Name())
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	orient, err := extractJPEGOrientation(outBytes)
	if err != nil {
		t.Fatalf("extractJPEGOrientation failed: %v", err)
	}
	if orient != 1 {
		t.Fatalf("expected orientation 1 after save, got %d", orient)
	}
	_ = origSegs
}
