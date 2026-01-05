package cli

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// EXIF represents a parsed subset of EXIF metadata with typed fields.
type EXIF struct {
	Make             string            `json:"make,omitempty"`
	Model            string            `json:"model,omitempty"`
	Software         string            `json:"software,omitempty"`
	Orientation      int               `json:"orientation,omitempty"`
	DateTime         string            `json:"datetime,omitempty"`
	DateTimeOriginal string            `json:"datetime_original,omitempty"`
	ExposureTime     string            `json:"exposure_time,omitempty"` // original "num/den"
	Exposure         float64           `json:"exposure,omitempty"`      // seconds
	ShutterSpeed     string            `json:"shutter_speed,omitempty"`
	ApertureValue    float64           `json:"aperture_value,omitempty"`
	FNumber          float64           `json:"f_number,omitempty"`
	MeteringMode     int               `json:"metering_mode,omitempty"`
	Flash            int               `json:"flash,omitempty"`
	ISOSpeed         int               `json:"iso,omitempty"`
	FocalLength      float64           `json:"focal_length_mm,omitempty"`
	LensModel        string            `json:"lens_model,omitempty"`
	GPS              *GPSData          `json:"gps,omitempty"`
	Raw              map[uint32]string `json:"raw,omitempty"`
}

// GPSData holds parsed GPS coordinates and raw GPS tag values.
type GPSData struct {
	Latitude     float64           `json:"lat,omitempty"`
	Longitude    float64           `json:"lon,omitempty"`
	LatRef       string            `json:"lat_ref,omitempty"`
	LonRef       string            `json:"lon_ref,omitempty"`
	Altitude     float64           `json:"altitude,omitempty"`
	AltitudeRef  int               `json:"alt_ref,omitempty"`
	GPSTimeStamp string            `json:"gps_time_stamp,omitempty"`
	GPSDateStamp string            `json:"gps_date_stamp,omitempty"`
	Timestamp    time.Time         `json:"timestamp,omitempty"`
	Raw          map[uint16]string `json:"raw,omitempty"`
}

const (
	ifdType0    = 0
	ifdTypeExif = 1
	ifdTypeGPS  = 2
)

// ExtractEXIFStruct reads JPEG file at path and returns a typed EXIF struct.
func ExtractEXIFStruct(path string) (EXIF, error) {
	var out EXIF
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if len(b) < 3 || !bytes.Equal(b[:3], []byte{0xFF, 0xD8, 0xFF}) {
		return out, fmt.Errorf("unsupported format for EXIF extraction")
	}
	tiffStart, err := parseTIFFStartFromJPEG(b)
	if err != nil {
		return out, err
	}
	tags, err := readEXIFTags(b, tiffStart)
	if err != nil {
		return out, err
	}
	out = convertTagsToEXIF(tags)
	return out, nil
}

// convertTagsToEXIF converts the keyed tag map into a typed EXIF struct.
func convertTagsToEXIF(tags map[uint32]string) EXIF {
	out := EXIF{Raw: map[uint32]string{}}
	for k, v := range tags {
		out.Raw[k] = v
	}
	get := func(ifd int, tag uint16) (string, bool) {
		key := (uint32(ifd) << 16) | uint32(tag)
		v, ok := tags[key]
		return v, ok
	}
	// IFD0
	if v, ok := get(ifdType0, 0x010F); ok { // Make
		out.Make = v
	}
	if v, ok := get(ifdType0, 0x0110); ok { // Model
		out.Model = v
	}
	if v, ok := get(ifdType0, 0x0112); ok { // Orientation
		if vi, err := strconv.Atoi(v); err == nil {
			out.Orientation = vi
		}
	}
	if v, ok := get(ifdType0, 0x0132); ok {
		out.DateTime = v
	}
	// ExifIFD
	if v, ok := get(ifdTypeExif, 0x829A); ok { // ExposureTime
		out.ExposureTime = v
		if f, err := parseRational(v); err == nil {
			out.Exposure = f
		}
	}
	if v, ok := get(ifdTypeExif, 0x829D); ok { // FNumber
		if f, err := parseRational(v); err == nil {
			out.FNumber = f
		}
	}
	if v, ok := get(ifdTypeExif, 0x9201); ok { // ShutterSpeedValue (RATIONAL)
		out.ShutterSpeed = v
	}
	if v, ok := get(ifdTypeExif, 0x9202); ok { // ApertureValue (RATIONAL)
		if f, err := parseRational(v); err == nil {
			out.ApertureValue = f
		}
	}
	if v, ok := get(ifdTypeExif, 0x9207); ok { // MeteringMode SHORT
		if iv, err := strconv.Atoi(strings.SplitN(v, ",", 2)[0]); err == nil {
			out.MeteringMode = iv
		}
	}
	if v, ok := get(ifdTypeExif, 0x9209); ok { // Flash SHORT
		if iv, err := strconv.Atoi(strings.SplitN(v, ",", 2)[0]); err == nil {
			out.Flash = iv
		}
	}
	if v, ok := get(ifdTypeExif, 0x8827); ok { // ISOSpeedRatings
		if iv, err := strconv.Atoi(strings.SplitN(v, ",", 2)[0]); err == nil {
			out.ISOSpeed = iv
		}
	}
	if v, ok := get(ifdTypeExif, 0x920A); ok { // FocalLength
		if f, err := parseRational(v); err == nil {
			out.FocalLength = f
		}
	}
	if v, ok := get(ifdTypeExif, 0x9003); ok { // DateTimeOriginal
		out.DateTimeOriginal = v
	}
	if v, ok := get(ifdType0, 0x0131); ok { // Software
		out.Software = v
	}
	if v, ok := get(ifdTypeExif, 0xA434); ok { // LensModel
		out.LensModel = v
	}
	// GPS
	latRef, _ := get(ifdTypeGPS, 0x0001)
	lonRef, _ := get(ifdTypeGPS, 0x0003)
	latVal, latOk := get(ifdTypeGPS, 0x0002)
	lonVal, lonOk := get(ifdTypeGPS, 0x0004)
	altRefVal, _ := get(ifdTypeGPS, 0x0005)
	altVal, altOk := get(ifdTypeGPS, 0x0006)
	timeVal, timeOk := get(ifdTypeGPS, 0x0007)
	dateVal, dateOk := get(ifdTypeGPS, 0x001D)
	if latOk && lonOk {
		latRats, err1 := parseRationalList(latVal)
		lonRats, err2 := parseRationalList(lonVal)
		if err1 == nil && err2 == nil {
			lat, err3 := gpsToDecimal(latRats, latRef)
			lon, err4 := gpsToDecimal(lonRats, lonRef)
			if err3 == nil && err4 == nil {
				out.GPS = &GPSData{Latitude: lat, Longitude: lon, LatRef: latRef, LonRef: lonRef, Raw: map[uint16]string{}}
				if v, ok := get(ifdTypeGPS, 0x0001); ok {
					out.GPS.Raw[0x0001] = v
				}
				if v, ok := get(ifdTypeGPS, 0x0002); ok {
					out.GPS.Raw[0x0002] = v
				}
				if v, ok := get(ifdTypeGPS, 0x0003); ok {
					out.GPS.Raw[0x0003] = v
				}
				if v, ok := get(ifdTypeGPS, 0x0004); ok {
					out.GPS.Raw[0x0004] = v
				}
				// Altitude
				if altOk {
					if rats, err := parseRationalList(altVal); err == nil && len(rats) > 0 {
						out.GPS.Altitude = rats[0]
					}
				}
				if altRefVal != "" {
					if iv, err := strconv.Atoi(altRefVal); err == nil {
						out.GPS.AltitudeRef = iv
					}
				}
				if timeOk {
					out.GPS.GPSTimeStamp = timeVal
				}
				if dateOk {
					out.GPS.GPSDateStamp = dateVal
				}
				// Parse GPS date+time into Timestamp if both present
				if timeOk && dateOk {
					if timeParts, err := parseRationalList(timeVal); err == nil && len(timeParts) >= 3 {
						// dateVal expected format "YYYY:MM:DD"
						parts := strings.Split(dateVal, ":")
						if len(parts) >= 3 {
							year, _ := strconv.Atoi(parts[0])
							month, _ := strconv.Atoi(parts[1])
							day, _ := strconv.Atoi(parts[2])
							hour := int(timeParts[0])
							min := int(timeParts[1])
							sec := int(timeParts[2])
							out.GPS.Timestamp = time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
						}
					}
				}
			}
		}
	}
	return out
}

// HasGPS reports whether EXIF contains GPS coordinates.
func (e EXIF) HasGPS() bool {
	return e.GPS != nil
}

// GPSLatLong returns the GPS latitude and longitude if present.
// The boolean indicates presence.
func (e EXIF) GPSLatLong() (lat, lon float64, ok bool) {
	if e.GPS == nil {
		return 0, 0, false
	}
	return e.GPS.Latitude, e.GPS.Longitude, true
}

// HasCoords reports whether GPSData contains coordinates.
func (g GPSData) HasCoords() bool {
	return !(g.Latitude == 0 && g.Longitude == 0)
}

// LatLong returns latitude and longitude.
func (g GPSData) LatLong() (float64, float64) {
	return g.Latitude, g.Longitude
}
