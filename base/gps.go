// Copyright © 2023 Sloan Childers
package base

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/adrianmo/go-nmea"
	dectofrac "github.com/av-elier/go-decimal-to-rational"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/rs/zerolog/log"
	"go.bug.st/serial"
)

type ExifInfo struct {
	tm           time.Time
	latitudeRef  string
	latitude     []exifcommon.Rational
	longitudeRef string
	longitude    []exifcommon.Rational
	trackRef     string
	track        exifcommon.Rational
	speedRef     string
	speed        exifcommon.Rational
}

type GPS struct {
	port  serial.Port
	rate  int
	nmea  nmea.RMC
	buf   []byte
	mutex sync.Mutex
}

func NewGPS(rate int) *GPS {
	return &GPS{rate: rate, buf: make([]byte, 1024)}
}

func (x *GPS) Open() error {
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open("/dev/ttyACM0", mode)
	if err != nil {
		log.Error().Err(err).Msg("serial.Open(/dev/ttyACM0)")
		x.port = nil
		return err
	}
	x.port = port
	return nil
}

func (x *GPS) ToNMEA() nmea.RMC {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	return x.nmea
}

func (x *GPS) ToExif() (*ExifInfo, error) {

	exif := &ExifInfo{}

	if x.port == nil {
		// NOTE:  this is a hack to run without a GPS sensor attached
		exif.latitude = GpsDegrees(2040.52614)
		exif.latitudeRef = "N"

		exif.longitude = GpsDegrees(10512.19292)
		exif.longitudeRef = "W"
		return exif, nil
	}

	x.mutex.Lock()
	defer x.mutex.Unlock()

	if false {
		// TODO:  tracks and speed aren't writing into the Exif data properly
		exif.trackRef = "T" // M magnetic north or T true north
		frac := dectofrac.NewRatP(x.nmea.Course, 0.01)
		exif.track = exifcommon.Rational{Numerator: uint32(frac.Num().Uint64()), Denominator: uint32(frac.Denom().Uint64())}

		exif.speedRef = "K"
		frac = dectofrac.NewRatP(x.nmea.Speed, 0.01)
		exif.speed = exifcommon.Rational{Numerator: uint32(frac.Num().Uint64()), Denominator: uint32(frac.Denom().Uint64())}
		tmString := fmt.Sprintf("20%d-%02d-%02dT%02d:%02d:%02dZ",
			x.nmea.Date.YY,
			x.nmea.Date.MM,
			x.nmea.Date.DD,
			x.nmea.Time.Hour,
			x.nmea.Time.Minute,
			x.nmea.Time.Second)

		var err error
		exif.tm, err = time.Parse("2006-01-02T15:04:05Z", tmString)
		if err != nil {
			log.Error().Err(err).Str("component", "gps").Str("time", tmString).Msg("exif parse time")
			return nil, err
		}
	}

	exif.latitude = GpsDegrees(x.nmea.Latitude)
	exif.latitudeRef = x.nmea.Fields[3]

	exif.longitude = GpsDegrees(x.nmea.Longitude)
	exif.longitudeRef = x.nmea.Fields[5]

	return exif, nil
}

func (x *GPS) ToJSON() string {
	data, _ := json.MarshalIndent(x.nmea, "", "    ")
	return string(data)
}

func (x *GPS) ToDMS(l float64) string {
	return nmea.FormatDMS(l)
}

func (x *GPS) Start() {
	if x.port != nil {
		go x.run()
	}
}

func (x *GPS) run() {

	for {
		n, err := x.port.Read(x.buf)
		if err != nil {
			log.Error().Err(err).Msg("port.Read")
			break
		}
		if strings.HasPrefix(string(x.buf[:n]), "$GNRMC") {
			scanner := bufio.NewScanner(bytes.NewReader(x.buf[:n]))
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				data := scanner.Bytes()
				s, err := nmea.Parse(string(data))
				if err != nil {
					log.Error().Err(err).Msg("nmea.Parse")
					continue
				}
				if s.DataType() == nmea.TypeRMC {
					x.mutex.Lock()
					x.nmea = s.(nmea.RMC)
					x.mutex.Unlock()
					if false {
						fmt.Println(x.ToJSON())
						fmt.Println(x.ToDMS(x.nmea.Latitude))
						fmt.Println(x.ToDMS(x.nmea.Longitude))

					}
				}
			}
		}
		time.Sleep(time.Duration(x.rate) * time.Second)
	}
}
