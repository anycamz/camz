// Copyright © 2023 Sloan Childers
package base

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	exifcommon "github.com/dsoprea/go-exif/v3/common"
	jis "github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/stretchr/testify/assert"
)

func TestWriteExif(t *testing.T) {
	// Create a GPSInfo struct.
	gpsInfo := ExifInfo{
		trackRef:     "T",
		track:        exifcommon.Rational{Numerator: 100, Denominator: 1},
		speedRef:     "K",
		speed:        exifcommon.Rational{Numerator: 200, Denominator: 1},
		latitudeRef:  "N",
		latitude:     GpsDegrees(37.337),
		longitudeRef: "W",
		longitude:    GpsDegrees(-122.418),
	}

	// Create an artist, make, model, and host string.
	artist := "Sloan Childers"
	make := "Apple"
	model := "iPhone 13 Pro"
	host := "localhost"

	// Create a time.Time object.
	jpegTime := time.Now()

	// Create a []byte of JPEG data.
	jpeg := EmptyFrame(320, 240)

	// Call the WriteExif function.
	newJpeg, err := WriteExif(&gpsInfo, artist, make, model, host, jpegTime, jpeg)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the new []byte is not equal to the old []byte.
	if bytes.Equal(newJpeg, jpeg) {
		t.Fatal("The new []byte is equal to the old []byte.")
	}

	// Check that the EXIF data in the new []byte is correct.
	ec, _ := jis.NewJpegMediaParser().ParseBytes(newJpeg)
	root, _, err := ec.Exif()
	assert.NoError(t, err)
	results, _ := root.FindTagWithName("Artist")
	artistOut, err := results[0].GetRawBytes()
	assert.NoError(t, err)
	assert.Equal(t, artist+"\x00", string(artistOut))
	// TODO:  check other exif fields
}

func TestGpsDegrees(t *testing.T) {

	// Test 1: Positive latitude
	expected := []exifcommon.Rational{
		{Numerator: 37, Denominator: 1},
		{Numerator: 46, Denominator: 1},
		{Numerator: 29, Denominator: 1},
	}
	actual := GpsDegrees(37.775)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}

	// Test 2: Negative latitude
	expected = []exifcommon.Rational{
		{Numerator: 37, Denominator: 1},
		{Numerator: 46, Denominator: 1},
		{Numerator: 29, Denominator: 1},
	}
	actual = GpsDegrees(-37.775)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}

	// Test 3: Positive longitude
	expected = []exifcommon.Rational{
		{Numerator: 122, Denominator: 1},
		{Numerator: 25, Denominator: 1},
		{Numerator: 4, Denominator: 1},
	}
	actual = GpsDegrees(122.418)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}

	// Test 4: Negative longitude
	expected = []exifcommon.Rational{
		{Numerator: 122, Denominator: 1},
		{Numerator: 25, Denominator: 1},
		{Numerator: 4, Denominator: 1},
	}
	actual = GpsDegrees(-122.418)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}

}
