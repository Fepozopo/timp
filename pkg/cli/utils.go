package cli

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Fepozopo/timp/pkg/stdimg"
)

// PromptLine displays a prompt and reads a full line of input from the user.
// The returned string is trimmed of surrounding whitespace (including the newline).
func PromptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// PromptLineOrFzf reads a full line from stdin and treats a single-line "/"
// as a request to invoke fzf for file selection. Behavior:
//   - Print the prompt.
//   - Read a full line (including spaces).
//   - If the trimmed line equals "/", launch fzf via SelectFileWithFzf(".").
//   - If fzf returns a non-empty selection, return it.
//   - If fzf is unavailable or selection is cancelled, fall back to a typed prompt
//     (re-using PromptLine to read a full line).
//   - Otherwise return the trimmed line as the input value.
//
// This approach preserves support for paths containing spaces because we read
// the entire input line instead of a single token.
func PromptLineOrFzf(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input := strings.TrimSpace(line)

	if input == "/" {
		// User requested fzf selection.
		sel, selErr := SelectFileWithFzf(".")
		if selErr == nil && sel != "" {
			// Show concise indicator and return the selection.
			fmt.Printf(" [fzf] %s\n", sel)
			return sel, nil
		}
		// fzf not available or selection cancelled — fall back to typed prompt.
		return PromptLine(prompt)
	}

	return input, nil
}

// PromptLineWithFzfReader is a convenience variant that reads from the provided
// bufio.Reader. This is useful when the caller already has a reader instance
// and wants to avoid creating a new one (ensures no input is lost to a
// separate buffered reader).
func PromptLineWithFzfReader(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input := strings.TrimSpace(line)

	if input == "/" {
		sel, selErr := SelectFileWithFzf(".")
		if selErr == nil && sel != "" {
			fmt.Printf(" [fzf] %s\n", sel)
			return sel, nil
		}
		return PromptLine(prompt)
	}
	return input, nil
}

// PromptLineWithFzf kept for backward compatibility; it delegates to
// PromptLineOrFzf (which reads the whole line and treats "/" as fzf trigger).
func PromptLineWithFzf(prompt string) (string, error) {
	return PromptLineOrFzf(prompt)
}

// LoadImage loads a file from disk into an image.Image. Supports PNG/JPEG/GIF based on file extension and decoders.
func LoadImage(path string) (image.Image, string, error) {
	// Read full file to allow EXIF inspection for JPEG orientation.
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	// quick format detection via magic
	format := ""
	if len(b) >= 3 && bytes.Equal(b[:3], []byte{0xFF, 0xD8, 0xFF}) {
		format = "jpeg"
	} else if len(b) >= 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")) {
		format = "png"
	} else if len(b) >= 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))) {
		format = "gif"
	}
	// If JPEG, try to extract EXIF orientation
	orientation := 1
	if format == "jpeg" {
		if o, err := extractJPEGOrientation(b); err == nil && o >= 1 && o <= 8 {
			orientation = o
		}
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	// apply auto-orient if needed
	if orientation != 1 {
		img = stdimg.AutoOrient(img, orientation)
	}
	if format == "" {
		// fallback to decoded format if we couldn't heuristically detect
		// the file signature
		// attempt to determine from decode
		// (image.Decode returned an image but not format; we already called Decode so "_" ignored)
		format = ""
	}
	return img, format, nil
}

// parseEXIFFromJPEG scans JPEG segments to find an APP1 Exif block and returns
// the TIFF start offset (index in data) where the TIFF header begins, or -1 if not found.
func parseTIFFStartFromJPEG(data []byte) (int, error) {
	if len(data) < 4 {
		return -1, fmt.Errorf("data too short")
	}
	i := 2 // skip initial 0xFF 0xD8
	for i+4 < len(data) {
		if data[i] != 0xFF {
			i++
			continue
		}
		marker := data[i+1]
		if marker == 0xDA { // start of scan
			break
		}
		if i+4 > len(data) {
			break
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		if marker == 0xE1 && segLen >= 8 {
			// check for "Exif\0\0"
			if i+4+6 <= len(data) && string(data[i+4:i+10]) == "Exif\x00\x00" {
				return i + 10, nil
			}
		}
		if segLen <= 2 {
			i += 2
		} else {
			i += 2 + segLen
		}
	}
	return -1, fmt.Errorf("no exif segment")
}

// readEXIFTags reads a set of tags from TIFF data starting at tiffStart.
// It returns a map where the key encodes the IFD type in the high 16 bits
// and the tag ID in the low 16 bits: (ifdType<<16)|tag. IFD types: 0=IFD0,1=ExifIFD,2=GPS.
// The function follows ExifIFD (tag 0x8769) and GPS IFD (tag 0x8825) pointers automatically.
func readEXIFTags(data []byte, tiffStart int) (map[uint32]string, error) {
	res := map[uint32]string{}
	if tiffStart+8 > len(data) {
		return res, fmt.Errorf("tiff header truncated")
	}
	// determine byte order
	var order binary.ByteOrder
	if data[tiffStart] == 'M' && data[tiffStart+1] == 'M' {
		order = binary.BigEndian
	} else if data[tiffStart] == 'I' && data[tiffStart+1] == 'I' {
		order = binary.LittleEndian
	} else {
		return res, fmt.Errorf("unknown tiff byte order")
	}
	// magic
	magic := order.Uint16(data[tiffStart+2 : tiffStart+4])
	if magic != 0x002A {
		return res, fmt.Errorf("invalid tiff magic")
	}

	const (
		ifdType0    = 0
		ifdTypeExif = 1
		ifdTypeGPS  = 2
	)

	visited := map[int]bool{}
	var readIFD func(ifdOffset int, ifdType int) error
	readIFD = func(ifdOffset int, ifdType int) error {
		absIfd := tiffStart + ifdOffset
		if absIfd+2 > len(data) {
			return fmt.Errorf("ifd truncated")
		}
		if visited[absIfd] {
			return nil
		}
		visited[absIfd] = true
		nEntries := int(order.Uint16(data[absIfd : absIfd+2]))
		entriesBase := absIfd + 2
		for e := 0; e < nEntries; e++ {
			ent := entriesBase + e*12
			if ent+12 > len(data) {
				break
			}
			tag := order.Uint16(data[ent : ent+2])
			typ := order.Uint16(data[ent+2 : ent+4])
			count := order.Uint32(data[ent+4 : ent+8])
			valOff := data[ent+8 : ent+12]
			// type sizes per TIFF: 1=BYTE(1),2=ASCII(1),3=SHORT(2),4=LONG(4),5=RATIONAL(8)
			sizePer := 1
			switch typ {
			case 1, 2:
				sizePer = 1
			case 3:
				sizePer = 2
			case 4:
				sizePer = 4
			case 5:
				sizePer = 8
			default:
				// unsupported types are skipped but continue scanning
				sizePer = 0
			}
			// helper to read raw bytes for the value
			var valueBytes []byte
			if sizePer == 0 {
				// unsupported; still check for pointers to follow Exif/GPS
				if tag == 0x8769 || tag == 0x8825 {
					off32 := int(order.Uint32(valOff))
					if off32 > 0 && tiffStart+off32 < len(data) {
						if tag == 0x8769 {
							_ = readIFD(off32, ifdTypeExif)
						} else {
							_ = readIFD(off32, ifdTypeGPS)
						}
					}
				}
				continue
			}
			totalSize := int(count) * sizePer
			if totalSize <= 4 {
				buf := make([]byte, 4)
				copy(buf, valOff)
				valueBytes = buf[:totalSize]
			} else {
				off32 := int(order.Uint32(valOff))
				if off32 < 0 || tiffStart+off32+totalSize > len(data) {
					continue
				}
				valueBytes = data[tiffStart+off32 : tiffStart+off32+totalSize]
			}
			// Special-case pointers to other IFDs: ExifIFD (0x8769) and GPS IFD (0x8825)
			if tag == 0x8769 || tag == 0x8825 {
				off32 := int(order.Uint32(valOff))
				if off32 > 0 && tiffStart+off32 < len(data) {
					if tag == 0x8769 {
						_ = readIFD(off32, ifdTypeExif)
					} else {
						_ = readIFD(off32, ifdTypeGPS)
					}
				}
				continue
			}
			// Decode based on type
			sval := ""
			switch typ {
			case 1: // BYTE
				if len(valueBytes) == 1 {
					sval = fmt.Sprintf("%d", valueBytes[0])
				} else {
					vals := make([]string, 0, len(valueBytes))
					for _, b := range valueBytes {
						vals = append(vals, fmt.Sprintf("%d", b))
					}
					sval = strings.Join(vals, ",")
				}
			case 2: // ASCII
				str := string(valueBytes)
				if idx := bytes.IndexByte(valueBytes, 0); idx >= 0 {
					str = string(valueBytes[:idx])
				}
				sval = str
			case 3: // SHORT (2 bytes)
				vals := make([]string, 0, count)
				for i := 0; i < int(count); i++ {
					off := i * 2
					if off+2 > len(valueBytes) {
						break
					}
					v := order.Uint16(valueBytes[off : off+2])
					vals = append(vals, fmt.Sprintf("%d", v))
				}
				sval = strings.Join(vals, ",")
			case 4: // LONG (4 bytes)
				vals := make([]string, 0, count)
				for i := 0; i < int(count); i++ {
					off := i * 4
					if off+4 > len(valueBytes) {
						break
					}
					v := order.Uint32(valueBytes[off : off+4])
					vals = append(vals, fmt.Sprintf("%d", v))
				}
				sval = strings.Join(vals, ",")
			case 5: // RATIONAL (8 bytes: two LONGs)
				vals := make([]string, 0, count)
				for i := 0; i < int(count); i++ {
					off := i * 8
					if off+8 > len(valueBytes) {
						break
					}
					num := order.Uint32(valueBytes[off : off+4])
					den := order.Uint32(valueBytes[off+4 : off+8])
					if den == 0 {
						vals = append(vals, fmt.Sprintf("%d/0", num))
					} else {
						vals = append(vals, fmt.Sprintf("%d/%d", num, den))
					}
				}
				sval = strings.Join(vals, ",")
			default:
				// unsupported types were handled earlier
			}
			// store with encoded key
			key := (uint32(ifdType) << 16) | uint32(tag)
			if sval != "" {
				res[key] = sval
			}
		}
		// follow next IFD offset if present (not strictly necessary for tags we care about)
		// but avoid infinite loops via visited map
		// The next IFD pointer is located immediately after the last entry
		last := entriesBase + nEntries*12
		if last+4 <= len(data) {
			nextOff := int(order.Uint32(data[last : last+4]))
			if nextOff > 0 && tiffStart+nextOff < len(data) {
				_ = readIFD(nextOff, ifdType)
			}
		}
		return nil
	}
	// start from 0th IFD offset
	off := int(order.Uint32(data[tiffStart+4 : tiffStart+8]))
	if off <= 0 || tiffStart+off >= len(data) {
		return res, nil
	}
	_ = readIFD(off, ifdType0)
	return res, nil
}

// parseRational parses a single "num/den" string into float64.
func parseRational(s string) (float64, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid rational: %s", s)
	}
	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}
	den, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}
	if den == 0 {
		return 0, fmt.Errorf("zero denominator")
	}
	return num / den, nil
}

// parseRationalList parses comma-separated rationals into floats.
func parseRationalList(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := parseRational(p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// gpsToDecimal converts 3-element degrees/minutes/seconds to decimal degrees, applying ref (N/S/E/W).
func gpsToDecimal(vals []float64, ref string) (float64, error) {
	if len(vals) < 1 {
		return 0, fmt.Errorf("empty gps values")
	}
	deg := vals[0]
	min := 0.0
	sec := 0.0
	if len(vals) >= 2 {
		min = vals[1]
	}
	if len(vals) >= 3 {
		sec = vals[2]
	}
	d := deg + min/60.0 + sec/3600.0
	if ref == "S" || ref == "W" {
		d = -d
	}
	return d, nil
}

// extractJPEGOrientation returns the EXIF orientation (1..8) from JPEG bytes.
func extractJPEGOrientation(data []byte) (int, error) {
	tiffStart, err := parseTIFFStartFromJPEG(data)
	if err != nil {
		return 0, err
	}
	tags, err := readEXIFTags(data, tiffStart)
	if err != nil {
		return 0, err
	}
	// search for orientation tag across IFDs
	for k, v := range tags {
		tag := uint16(k & 0xffff)
		if tag == 0x0112 {
			if vi, err := strconv.Atoi(v); err == nil {
				return vi, nil
			}
		}
	}
	return 0, fmt.Errorf("orientation tag not found")
}

// SaveImage saves an image.Image to disk using format inferred from the filename extension.
// Supports .png, .jpg/.jpeg, .gif
func SaveImage(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return png.Encode(f, img)
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 92})
	case ".gif":
		return gif.Encode(f, img, nil)
	default:
		// default to PNG
		return png.Encode(f, img)
	}
}

// GetImageInfoImage returns a short info string for an image.Image
func GetImageInfoImage(img image.Image) (string, error) {
	if img == nil {
		return "", fmt.Errorf("nil image")
	}
	b := img.Bounds()
	format := "unknown"
	switch img.(type) {
	case *image.YCbCr:
		format = "JPEG"
	case *image.Paletted:
		format = "GIF"
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64,
		*image.Gray, *image.Gray16, *image.Alpha, *image.Alpha16, *image.Uniform:
		// Most non-JPEG/non-GIF decoded images are typically PNG (or other raster formats).
		// We default to PNG as the most common lossless container for these types.
		format = "PNG"
	default:
		// leave as "unknown" if we can't heuristically determine it
	}
	return fmt.Sprintf("Format: %s, Width: %d, Height: %d", format, b.Dx(), b.Dy()), nil
}

// PreviewImage wrapper will be provided in terminal_preview.go which knows how to send PNG bytes to the terminal.
