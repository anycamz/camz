// Copyright © 2023 Sloan Childers
package opencv

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/blackjack/webcam"
	"github.com/osintami/camz/base"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"
)

// "uuid": "<uuid>",
// "plugin": "opencv",
// "device": 0,
// "enabled": true,
// "name": "Main",
// "width": 320,
// "height": 240,
// "rate": 20,

type Driver struct {
	config *base.CameraConfig
	webcam *gocv.VideoCapture
	frame  gocv.Mat
	mutex  sync.Mutex
	stop   bool
}

func NewDriver(config *base.CameraConfig) base.IDriver {
	return &Driver{
		config: config,
		frame:  gocv.NewMat(),
	}
}

func (x *Driver) ListFormatsAndFrameSizes() base.Formats {
	formats := base.Formats{}
	webcam, err := webcam.Open(fmt.Sprintf("/dev/video%d", x.config.Device))
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Int("device", x.config.Device).Msg("webcam.Open")
		return formats
	}
	format_desc := webcam.GetSupportedFormats()
	for formatObj, formatStr := range format_desc {
		format := base.Format{Name: formatStr}
		sizes := []base.Size{}
		fs := base.FrameSizes(webcam.GetSupportedFrameSizes(formatObj))
		sort.Sort(fs)
		for _, frameSize := range fs {
			sizes = append(sizes, base.Size{Size: frameSize.GetString()})
		}
		format.Sizes = sizes
		formats.Formats = append(formats.Formats, format)
	}
	webcam.Close()
	return formats
}

func (x *Driver) Grab() base.IFrame {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	frame := base.NewFrame(x.config)
	frame.SetImage(x.frame.Clone(), base.GOCV)
	return frame
}

func (x *Driver) grab() gocv.Mat {
	img := gocv.NewMat()
	if !x.webcam.Read(&img) {
		if !x.webcam.Read(&img) {
			if !x.webcam.Read(&img) {
				log.Warn().Str("component", "driver").Str("name", x.config.Name).Msg("empty frame")
				jpeg := base.EmptyFrame(x.config.Width, x.config.Height)
				err := gocv.IMDecodeIntoMat(jpeg, gocv.IMReadAnyColor, &img)
				if err != nil {
					log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("JPEG decode with OpenCV")
				}
			}
		}
	}
	return img
}

func (x *Driver) Stream() {
	go x.stream()
}

func (x *Driver) Open() error {
	var err error
	x.webcam, err = gocv.VideoCaptureDevice(x.config.Device)
	if err != nil {
		log.Fatal().Err(err).Str("component", "webcam").Str("name", x.config.Name).Int("device", x.config.Device).Msg("webcam.VideoCaptureDevice")
		return err
	}
	x.webcam.Set(gocv.VideoCaptureFrameWidth, float64(x.config.Width))
	x.webcam.Set(gocv.VideoCaptureFrameHeight, float64(x.config.Height))
	x.webcam.Set(gocv.VideoCaptureFPS, float64(x.config.Rate))
	x.stop = false
	return nil
}

func (x *Driver) Stop() {
	x.mutex.Lock()
	x.stop = true
	err := x.webcam.Close()
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("Close")
	}
	x.mutex.Unlock()
}

func (x *Driver) Reset() error {
	x.Stop()
	err := x.Open()
	if err != nil {
		json, err := json.Marshal(x.config)
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).RawJSON("config", json).Msg("Open")
		return err
	}
	x.Stream()
	return nil
}
func (x *Driver) stream() {
	for {
		startTime := time.Now().UnixMilli()
		x.mutex.Lock()
		if x.stop {
			x.mutex.Unlock()
			return
		}
		x.frame.Close()
		x.frame = x.grab()
		// TODO:  investigate skipping frames in lieu of a higher framerate to avoid buffering
		//x.webcam.Grab(2)
		x.mutex.Unlock()
		base.Sleep(int64(x.config.Rate), time.Now().UnixMilli()-startTime)
	}
}
