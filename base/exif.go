// Copyright © 2023 Sloan Childers
package base

import (
	"bufio"
	"bytes"
	"math"
	"time"

	dsoprea "github.com/dsoprea/go-exif"
	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	jis "github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/rs/zerolog/log"
)

func WriteExif(gpsInfo *ExifInfo, artist, make, model, host string, jpegTime time.Time, jpeg []byte) ([]byte, error) {
	intfc, _ := jis.NewJpegMediaParser().ParseBytes(jpeg)
	sl := intfc.(*jis.SegmentList)
	ib, err := sl.ConstructExifBuilder()
	if err != nil {
		log.Error().Err(err).Str("component", "image").Msg("ConstructExifBuilder")
		return jpeg, err
	}

	ifd0Ib, _ := exif.GetOrCreateIbFromRootIb(ib, "IFD")
	exifIb, _ := exif.GetOrCreateIbFromRootIb(ib, dsoprea.IfdPathStandardExif)
	ifdGps, _ := exif.GetOrCreateIbFromRootIb(ib, dsoprea.IfdPathStandardGps)

	ifd0Ib.SetStandardWithName("Artist", artist)
	ifd0Ib.SetStandardWithName("Make", make)
	ifd0Ib.SetStandardWithName("Model", model)
	ifd0Ib.SetStandardWithName("HostComputer", host)

	//ifdGps.SetStandardWithName("GPSVersionID", []byte{2, 3, 0, 0})
	ifdGps.SetStandardWithName("GPSTrackRef", gpsInfo.trackRef)
	ifdGps.SetStandardWithName("GPSTrack", gpsInfo.track)
	ifdGps.SetStandardWithName("GPSSpeedRef", gpsInfo.speedRef)
	ifdGps.SetStandardWithName("GPSSpeed", gpsInfo.speed)
	ifdGps.SetStandardWithName("GPSLatitudeRef", gpsInfo.latitudeRef)
	ifdGps.SetStandardWithName("GPSLatitude", gpsInfo.latitude)
	ifdGps.SetStandardWithName("GPSLongitudeRef", gpsInfo.longitudeRef)
	ifdGps.SetStandardWithName("GPSLongitude", gpsInfo.longitude)
	//ifdGps.SetStandardWithName("GPSAltitudeRef", byte(0x00))
	//ifdGps.SetStandardWithName("GPSAltitude", exifcommon.Rational{Numerator: 517150, Denominator: 10321})

	exifIb.SetStandardWithName("DateTimeOriginal", jpegTime)
	_ = sl.SetExif(ib)

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	_ = sl.Write(w)
	w.Flush()

	data := Copy(buf.Bytes())
	return data, nil
}

// func DebugExif(jpeg []byte) {
// 	ec, _ := jis.NewJpegMediaParser().ParseBytes(jpeg)
// 	root, _, _ := ec.Exif()
// 	for entry := range root.Entries() {
// 		fmt.Println(entry)
// 	}
// }

func GpsDegrees(l float64) []exifcommon.Rational {
	val := math.Abs(l)
	degrees := int(math.Floor(val))
	minutes := int(math.Floor(60 * (val - float64(degrees))))
	seconds := 3600 * (val - float64(degrees) - (float64(minutes) / 60))
	return []exifcommon.Rational{
		{Numerator: uint32(degrees), Denominator: 1},
		{Numerator: uint32(minutes), Denominator: 1},
		{Numerator: uint32(seconds), Denominator: 1},
	}
}
