package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestEXIFGPSHelpers(t *testing.T) {
	b, err := buildJPEGWithEXIF()
	if err != nil {
		t.Fatalf("buildJPEGWithEXIF failed: %v", err)
	}
	f, err := os.CreateTemp("", "exif-fixture-*.jpg")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(b); err != nil {
		f.Close()
		t.Fatalf("write temp file failed: %v", err)
	}
	f.Close()

	ex, err := ExtractEXIFStruct(f.Name())
	if err != nil {
		t.Fatalf("ExtractEXIFStruct failed: %v", err)
	}

	if ex.Orientation != 6 {
		t.Fatalf("expected Orientation 6, got %d", ex.Orientation)
	}
	if math.Abs(ex.FocalLength-50.0) > 1e-9 {
		t.Fatalf("expected FocalLength 50, got %v", ex.FocalLength)
	}
	if ex.ISOSpeed != 100 {
		t.Fatalf("expected ISOSpeed 100, got %d", ex.ISOSpeed)
	}
	if math.Abs(ex.FNumber-5.0) > 1e-9 {
		t.Fatalf("expected FNumber 5, got %v", ex.FNumber)
	}
	if math.Abs(ex.Exposure-1.0/60.0) > 1e-9 {
		t.Fatalf("expected Exposure ~1/60, got %v", ex.Exposure)
	}
	if ex.DateTimeOriginal != "2020:01:02 03:04:05" {
		t.Fatalf("expected DateTimeOriginal, got %q", ex.DateTimeOriginal)
	}

	if !ex.HasGPS() {
		t.Fatalf("expected GPS present")
	}
	lat, lon, ok := ex.GPSLatLong()
	if !ok {
		t.Fatalf("expected GPSLatLong ok")
	}
	expLat := 37.0 + 48.0/60.0 + 30.0/3600.0
	expLon := -(122.0 + 24.0/60.0 + 15.0/3600.0)
	if math.Abs(lat-expLat) > 1e-9 {
		t.Fatalf("latitude mismatch: expected %v got %v", expLat, lat)
	}
	if math.Abs(lon-expLon) > 1e-9 {
		t.Fatalf("longitude mismatch: expected %v got %v", expLon, lon)
	}

	if ex.GPS == nil {
		t.Fatalf("expected ex.GPS not nil")
	}
	if !ex.GPS.HasCoords() {
		t.Fatalf("expected GPS.HasCoords true")
	}
	lat2, lon2 := ex.GPS.LatLong()
	if math.Abs(lat2-expLat) > 1e-9 || math.Abs(lon2-expLon) > 1e-9 {
		t.Fatalf("GPS.LatLong mismatch")
	}
}

// Big-endian TIFF variant test
func TestEXIFBigEndian(t *testing.T) {
	b, err := buildJPEGWithEXIFBigEndian()
	if err != nil {
		t.Fatalf("buildJPEGWithEXIFBigEndian failed: %v", err)
	}
	f, err := os.CreateTemp("", "exif-be-fixture-*.jpg")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(b); err != nil {
		f.Close()
		t.Fatalf("write temp file failed: %v", err)
	}
	f.Close()

	ex, err := ExtractEXIFStruct(f.Name())
	if err != nil {
		t.Fatalf("ExtractEXIFStruct failed: %v", err)
	}
	// Same expectations as little-endian case
	if ex.Orientation != 6 || ex.ISOSpeed != 100 || math.Abs(ex.FNumber-5.0) > 1e-9 {
		t.Fatalf("big-endian parsing mismatch: %+v", ex)
	}
	if !ex.HasGPS() {
		t.Fatalf("expected GPS present")
	}
}

// Malformed IFD pointer should not panic; result may be empty
func TestEXIFMalformedIFD(t *testing.T) {
	b, err := buildJPEGWithMalformedIFD()
	if err != nil {
		t.Fatalf("buildJPEGWithMalformedIFD failed: %v", err)
	}
	f, err := os.CreateTemp("", "exif-malformed-*.jpg")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(b); err != nil {
		f.Close()
		t.Fatalf("write temp file failed: %v", err)
	}
	f.Close()

	ex, err := ExtractEXIFStruct(f.Name())
	if err != nil {
		t.Fatalf("ExtractEXIFStruct returned error on malformed IFD: %v", err)
	}
	// Expect empty EXIF (no GPS, zero orientation)
	if ex.HasGPS() {
		t.Fatalf("expected no GPS for malformed IFD")
	}
	if ex.Orientation != 0 {
		t.Fatalf("expected Orientation 0 for malformed IFD, got %d", ex.Orientation)
	}
}

// buildJPEGWithEXIF builds a little-endian TIFF EXIF block similar to previous test.
func buildJPEGWithEXIF() ([]byte, error) {
	var tiff bytes.Buffer
	// TIFF header: II, 0x2A, offset to IFD0=8
	tiff.Write([]byte{'I', 'I'})
	binary.Write(&tiff, binary.LittleEndian, uint16(0x2A))
	binary.Write(&tiff, binary.LittleEndian, uint32(8))

	ifd0Count := uint16(4) // Orientation, ExifIFDPointer, GPSInfoIFDPointer, Software
	ifd0Len := int(2 + int(ifd0Count)*12 + 4)
	exifOffset := 8 + uint32(ifd0Len)

	exifCount := uint16(10) // exposure,fnumber,shutterspeed,aperture,metering,flash,iso,focal,dtorig,lensmodel
	exifIFDLen := int(2 + int(exifCount)*12 + 4)
	dataStart := exifOffset + uint32(exifIFDLen)
	dataBuf := bytes.Buffer{}

	binary.Write(&tiff, binary.LittleEndian, ifd0Count)
	type ifdEntry struct {
		tag, typeID  uint16
		count, value uint32
	}
	var ifd0Entries []ifdEntry
	// Orientation inline SHORT
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x0112, typeID: 3, count: 1, value: 6})
	// ExifIFDPointer (will point to exifOffset)
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x8769, typeID: 4, count: 1, value: exifOffset})
	// GPSInfoIFDPointer placeholder
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x8825, typeID: 4, count: 1, value: 0})
	// Software (ASCII) -> write later into dataBuf and point to it
	// placeholder value 0 for now; we'll patch after data is arranged
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x0131, typeID: 2, count: 0, value: 0})

	for _, e := range ifd0Entries {
		binary.Write(&tiff, binary.LittleEndian, e.tag)
		binary.Write(&tiff, binary.LittleEndian, e.typeID)
		binary.Write(&tiff, binary.LittleEndian, e.count)
		binary.Write(&tiff, binary.LittleEndian, e.value)
	}
	binary.Write(&tiff, binary.LittleEndian, uint32(0))

	if uint32(tiff.Len()) != exifOffset {
		return nil, fmt.Errorf("unexpected exifOffset mismatch: %d vs %d", tiff.Len(), exifOffset)
	}

	// Build Exif IFD entries
	binary.Write(&tiff, binary.LittleEndian, exifCount)
	var exifEntries []ifdEntry
	// ExposureTime (RATIONAL)
	exposureOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, uint32(1))
	binary.Write(&dataBuf, binary.LittleEndian, uint32(60))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x829A, typeID: 5, count: 1, value: exposureOffset})
	// FNumber (RATIONAL)
	fnumOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, uint32(5))
	binary.Write(&dataBuf, binary.LittleEndian, uint32(1))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x829D, typeID: 5, count: 1, value: fnumOffset})
	// ShutterSpeedValue (RATIONAL) example 600/100 -> 6/1
	ssOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, uint32(6))
	binary.Write(&dataBuf, binary.LittleEndian, uint32(1))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x9201, typeID: 5, count: 1, value: ssOffset})
	// ApertureValue (RATIONAL) example 28/10 -> 2.8
	apOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, uint32(28))
	binary.Write(&dataBuf, binary.LittleEndian, uint32(10))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x9202, typeID: 5, count: 1, value: apOffset})
	// MeteringMode (SHORT) inline = 5
	exifEntries = append(exifEntries, ifdEntry{tag: 0x9207, typeID: 3, count: 1, value: 5})
	// Flash (SHORT) inline = 1
	exifEntries = append(exifEntries, ifdEntry{tag: 0x9209, typeID: 3, count: 1, value: 1})
	// ISOSpeedRatings (SHORT) inline = 100
	exifEntries = append(exifEntries, ifdEntry{tag: 0x8827, typeID: 3, count: 1, value: 100})
	// FocalLength (RATIONAL)
	focalOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, uint32(50))
	binary.Write(&dataBuf, binary.LittleEndian, uint32(1))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x920A, typeID: 5, count: 1, value: focalOffset})
	// DateTimeOriginal (ASCII)
	dt := []byte("2020:01:02 03:04:05")
	dtOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, dt)
	binary.Write(&dataBuf, binary.LittleEndian, byte(0))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x9003, typeID: 2, count: uint32(len(dt) + 1), value: dtOffset})
	// LensModel (ASCII)
	lens := []byte("GoLensModel")
	lensOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.LittleEndian, lens)
	binary.Write(&dataBuf, binary.LittleEndian, byte(0))
	exifEntries = append(exifEntries, ifdEntry{tag: 0xA434, typeID: 2, count: uint32(len(lens) + 1), value: lensOffset})

	// Now write Exif entries
	for _, e := range exifEntries {
		binary.Write(&tiff, binary.LittleEndian, e.tag)
		binary.Write(&tiff, binary.LittleEndian, e.typeID)
		binary.Write(&tiff, binary.LittleEndian, e.count)
		binary.Write(&tiff, binary.LittleEndian, e.value)
	}
	binary.Write(&tiff, binary.LittleEndian, uint32(0))
	// Append exif data
	if _, err := tiff.Write(dataBuf.Bytes()); err != nil {
		return nil, err
	}

	// Software string for IFD0: will append after GPS data
	soft := []byte("GoTest")

	// GPS IFD
	gpsOffset := uint32(tiff.Len())
	// We'll build GPS entries with more tags: lat, lon, altRef, alt, timeStamp, dateStamp
	gpsCount := uint16(8)
	binary.Write(&tiff, binary.LittleEndian, gpsCount)
	var gpsData bytes.Buffer
	// latitude rationals
	binary.Write(&gpsData, binary.LittleEndian, uint32(37))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	binary.Write(&gpsData, binary.LittleEndian, uint32(48))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	binary.Write(&gpsData, binary.LittleEndian, uint32(30))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	// longitude rationals
	binary.Write(&gpsData, binary.LittleEndian, uint32(122))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	binary.Write(&gpsData, binary.LittleEndian, uint32(24))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	binary.Write(&gpsData, binary.LittleEndian, uint32(15))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	// altitude rational 100/1
	binary.Write(&gpsData, binary.LittleEndian, uint32(100))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	// GPSTimeStamp (3 rationals) 12:30:15 -> 12/1,30/1,15/1
	binary.Write(&gpsData, binary.LittleEndian, uint32(12))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	binary.Write(&gpsData, binary.LittleEndian, uint32(30))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	binary.Write(&gpsData, binary.LittleEndian, uint32(15))
	binary.Write(&gpsData, binary.LittleEndian, uint32(1))
	// GPSDateStamp string
	dateStamp := []byte("2020:01:02")
	binary.Write(&gpsData, binary.LittleEndian, dateStamp)
	binary.Write(&gpsData, binary.LittleEndian, byte(0))

	gpsDataStart := gpsOffset + uint32(2+int(gpsCount)*12+4)
	off := gpsDataStart
	latValsOff := off
	off += 3 * 8
	lonValsOff := off
	off += 3 * 8
	altValsOff := off
	off += 2 * 4
	timeValsOff := off
	off += 3 * 8
	dateValsOff := off
	off += uint32(len(dateStamp) + 1)

	// Build GPS entries
	// GPSLatitudeRef inline 'N'
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0001))
	binary.Write(&tiff, binary.LittleEndian, uint16(2))
	binary.Write(&tiff, binary.LittleEndian, uint32(2))
	binary.Write(&tiff, binary.LittleEndian, uint32('N'))
	// GPSLatitude
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0002))
	binary.Write(&tiff, binary.LittleEndian, uint16(5))
	binary.Write(&tiff, binary.LittleEndian, uint32(3))
	binary.Write(&tiff, binary.LittleEndian, latValsOff)
	// GPSLongitudeRef inline 'W'
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0003))
	binary.Write(&tiff, binary.LittleEndian, uint16(2))
	binary.Write(&tiff, binary.LittleEndian, uint32(2))
	binary.Write(&tiff, binary.LittleEndian, uint32('W'))
	// GPSLongitude
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0004))
	binary.Write(&tiff, binary.LittleEndian, uint16(5))
	binary.Write(&tiff, binary.LittleEndian, uint32(3))
	binary.Write(&tiff, binary.LittleEndian, lonValsOff)
	// GPSAltitudeRef (BYTE)
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0005))
	binary.Write(&tiff, binary.LittleEndian, uint16(1))
	binary.Write(&tiff, binary.LittleEndian, uint32(1))
	binary.Write(&tiff, binary.LittleEndian, uint32(0x00))
	// GPSAltitude (RATIONAL)
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0006))
	binary.Write(&tiff, binary.LittleEndian, uint16(5))
	binary.Write(&tiff, binary.LittleEndian, uint32(1))
	binary.Write(&tiff, binary.LittleEndian, altValsOff)
	// GPSTimeStamp (3 RATIONALS)
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0007))
	binary.Write(&tiff, binary.LittleEndian, uint16(5))
	binary.Write(&tiff, binary.LittleEndian, uint32(3))
	binary.Write(&tiff, binary.LittleEndian, timeValsOff)
	// GPSDateStamp (ASCII)
	binary.Write(&tiff, binary.LittleEndian, uint16(0x001D))
	binary.Write(&tiff, binary.LittleEndian, uint16(2))
	binary.Write(&tiff, binary.LittleEndian, uint32(len(dateStamp)+1))
	binary.Write(&tiff, binary.LittleEndian, dateValsOff)
	// next IFD pointer
	binary.Write(&tiff, binary.LittleEndian, uint32(0))
	// Append gpsData
	if _, err := tiff.Write(gpsData.Bytes()); err != nil {
		return nil, err
	}

	// After appending gpsData, Software offset is current length
	softOffset := uint32(tiff.Len())
	// Append software bytes
	if _, err := tiff.Write(soft); err != nil {
		return nil, err
	}
	if err := binary.Write(&tiff, binary.LittleEndian, byte(0)); err != nil {
		return nil, err
	}

	// Patch Software entry in IFD0: find position of 4th entry's count and value
	buf := tiff.Bytes()
	ifd0EntriesStart := 8 + 2
	softEntryIndex := 3 // zero-based index
	softCountPos := ifd0EntriesStart + softEntryIndex*12 + 4
	softValuePos := ifd0EntriesStart + softEntryIndex*12 + 8
	if int(softValuePos+4) > len(buf) {
		return nil, fmt.Errorf("softEntryPos out of range")
	}
	binary.LittleEndian.PutUint32(buf[softCountPos:softCountPos+4], uint32(len(soft)+1))
	binary.LittleEndian.PutUint32(buf[softValuePos:softValuePos+4], uint32(softOffset))

	// Patch GPSInfoIFDPointer in IFD0 (3rd entry)
	gpsEntryIndex := 2
	gpsEntryValuePos := ifd0EntriesStart + gpsEntryIndex*12 + 8
	if int(gpsEntryValuePos+4) > len(buf) {
		return nil, fmt.Errorf("gpsEntryValuePos out of range")
	}
	binary.LittleEndian.PutUint32(buf[gpsEntryValuePos:gpsEntryValuePos+4], gpsOffset)

	// Build final JPEG
	var out bytes.Buffer
	out.Write([]byte{0xFF, 0xD8})
	out.Write([]byte{0xFF, 0xE1})
	app1Len := uint16(2 + 6 + len(buf))
	binary.Write(&out, binary.BigEndian, app1Len)
	out.Write([]byte("Exif\x00\x00"))
	out.Write(buf)
	out.Write([]byte{0xFF, 0xD9})
	return out.Bytes(), nil
}

// buildJPEGWithEXIFBigEndian builds a big-endian TIFF EXIF block.
func buildJPEGWithEXIFBigEndian() ([]byte, error) {
	var tiff bytes.Buffer
	tiff.Write([]byte{'M', 'M'})
	binary.Write(&tiff, binary.BigEndian, uint16(0x2A))
	binary.Write(&tiff, binary.BigEndian, uint32(8))

	ifd0Count := uint16(3)
	ifd0Len := int(2 + int(ifd0Count)*12 + 4)
	exifOffset := 8 + uint32(ifd0Len)
	exifCount := uint16(5)
	exifIFDLen := int(2 + int(exifCount)*12 + 4)
	dataStart := exifOffset + uint32(exifIFDLen)
	dataBuf := bytes.Buffer{}

	binary.Write(&tiff, binary.BigEndian, ifd0Count)
	type ifdEntry struct {
		tag, typeID  uint16
		count, value uint32
	}
	var ifd0Entries []ifdEntry
	// For big-endian, inline SHORT must occupy the high-order bytes of the 4-byte field.
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x0112, typeID: 3, count: 1, value: uint32(6) << 16})
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x8769, typeID: 4, count: 1, value: exifOffset})
	ifd0Entries = append(ifd0Entries, ifdEntry{tag: 0x8825, typeID: 4, count: 1, value: 0})
	for _, e := range ifd0Entries {
		binary.Write(&tiff, binary.BigEndian, e.tag)
		binary.Write(&tiff, binary.BigEndian, e.typeID)
		binary.Write(&tiff, binary.BigEndian, e.count)
		binary.Write(&tiff, binary.BigEndian, e.value)
	}
	binary.Write(&tiff, binary.BigEndian, uint32(0))

	if uint32(tiff.Len()) != exifOffset {
		return nil, fmt.Errorf("unexpected exifOffset mismatch: %d vs %d", tiff.Len(), exifOffset)
	}

	binary.Write(&tiff, binary.BigEndian, exifCount)
	var exifEntries []ifdEntry
	// ExposureTime
	exposureOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.BigEndian, uint32(1))
	binary.Write(&dataBuf, binary.BigEndian, uint32(60))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x829A, typeID: 5, count: 1, value: exposureOffset})
	// FNumber
	fnumOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.BigEndian, uint32(5))
	binary.Write(&dataBuf, binary.BigEndian, uint32(1))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x829D, typeID: 5, count: 1, value: fnumOffset})
	// ISOSpeedRatings inline (for big-endian SHORT must be in high-order bytes)
	exifEntries = append(exifEntries, ifdEntry{tag: 0x8827, typeID: 3, count: 1, value: uint32(100) << 16})
	// FocalLength
	focalOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.BigEndian, uint32(50))
	binary.Write(&dataBuf, binary.BigEndian, uint32(1))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x920A, typeID: 5, count: 1, value: focalOffset})
	// DateTimeOriginal
	dt := []byte("2020:01:02 03:04:05")
	dtOffset := dataStart + uint32(dataBuf.Len())
	binary.Write(&dataBuf, binary.BigEndian, dt)
	binary.Write(&dataBuf, binary.BigEndian, byte(0))
	exifEntries = append(exifEntries, ifdEntry{tag: 0x9003, typeID: 2, count: uint32(len(dt) + 1), value: dtOffset})

	for _, e := range exifEntries {
		binary.Write(&tiff, binary.BigEndian, e.tag)
		binary.Write(&tiff, binary.BigEndian, e.typeID)
		binary.Write(&tiff, binary.BigEndian, e.count)
		binary.Write(&tiff, binary.BigEndian, e.value)
	}
	binary.Write(&tiff, binary.BigEndian, uint32(0))
	if _, err := tiff.Write(dataBuf.Bytes()); err != nil {
		return nil, err
	}

	gpsOffset := uint32(tiff.Len())
	gpsCount := uint16(4)
	binary.Write(&tiff, binary.BigEndian, gpsCount)
	var gpsData bytes.Buffer
	binary.Write(&gpsData, binary.BigEndian, uint32(37))
	binary.Write(&gpsData, binary.BigEndian, uint32(1))
	binary.Write(&gpsData, binary.BigEndian, uint32(48))
	binary.Write(&gpsData, binary.BigEndian, uint32(1))
	binary.Write(&gpsData, binary.BigEndian, uint32(30))
	binary.Write(&gpsData, binary.BigEndian, uint32(1))
	binary.Write(&gpsData, binary.BigEndian, uint32(122))
	binary.Write(&gpsData, binary.BigEndian, uint32(1))
	binary.Write(&gpsData, binary.BigEndian, uint32(24))
	binary.Write(&gpsData, binary.BigEndian, uint32(1))
	binary.Write(&gpsData, binary.BigEndian, uint32(15))
	binary.Write(&gpsData, binary.BigEndian, uint32(1))
	gpsDataStart := gpsOffset + uint32(2+int(gpsCount)*12+4)
	off := gpsDataStart
	latValsOff := off
	off += 3 * 8
	lonValsOff := off
	off += 3 * 8

	binary.Write(&tiff, binary.BigEndian, uint16(0x0001))
	binary.Write(&tiff, binary.BigEndian, uint16(2))
	binary.Write(&tiff, binary.BigEndian, uint32(2))
	binary.Write(&tiff, binary.BigEndian, uint32('N'))
	binary.Write(&tiff, binary.BigEndian, uint16(0x0002))
	binary.Write(&tiff, binary.BigEndian, uint16(5))
	binary.Write(&tiff, binary.BigEndian, uint32(3))
	binary.Write(&tiff, binary.BigEndian, latValsOff)
	binary.Write(&tiff, binary.BigEndian, uint16(0x0003))
	binary.Write(&tiff, binary.BigEndian, uint16(2))
	binary.Write(&tiff, binary.BigEndian, uint32(2))
	binary.Write(&tiff, binary.BigEndian, uint32('W'))
	binary.Write(&tiff, binary.BigEndian, uint16(0x0004))
	binary.Write(&tiff, binary.BigEndian, uint16(5))
	binary.Write(&tiff, binary.BigEndian, uint32(3))
	binary.Write(&tiff, binary.BigEndian, lonValsOff)
	binary.Write(&tiff, binary.BigEndian, uint32(0))
	if _, err := tiff.Write(gpsData.Bytes()); err != nil {
		return nil, err
	}

	buf := tiff.Bytes()
	ifd0EntriesStart := 8 + 2
	gpsEntryIndex := 2
	gpsEntryValuePos := ifd0EntriesStart + gpsEntryIndex*12 + 8
	if int(gpsEntryValuePos+4) > len(buf) {
		return nil, fmt.Errorf("gpsEntryValuePos out of range")
	}
	binary.BigEndian.PutUint32(buf[gpsEntryValuePos:gpsEntryValuePos+4], gpsOffset)

	var out bytes.Buffer
	out.Write([]byte{0xFF, 0xD8})
	out.Write([]byte{0xFF, 0xE1})
	app1Len := uint16(2 + 6 + len(buf))
	binary.Write(&out, binary.BigEndian, app1Len)
	out.Write([]byte("Exif\x00\x00"))
	out.Write(buf)
	out.Write([]byte{0xFF, 0xD9})
	return out.Bytes(), nil
}

// buildJPEGWithMalformedIFD builds a TIFF with an IFD0 offset that points beyond the buffer.
func buildJPEGWithMalformedIFD() ([]byte, error) {
	var tiff bytes.Buffer
	// Use little-endian header but bogus IFD offset
	tiff.Write([]byte{'I', 'I'})
	binary.Write(&tiff, binary.LittleEndian, uint16(0x2A))
	binary.Write(&tiff, binary.LittleEndian, uint32(0xFFFFFF))

	buf := tiff.Bytes()
	var out bytes.Buffer
	out.Write([]byte{0xFF, 0xD8})
	out.Write([]byte{0xFF, 0xE1})
	app1Len := uint16(2 + 6 + len(buf))
	binary.Write(&out, binary.BigEndian, app1Len)
	out.Write([]byte("Exif\x00\x00"))
	out.Write(buf)
	out.Write([]byte{0xFF, 0xD9})
	return out.Bytes(), nil
}
