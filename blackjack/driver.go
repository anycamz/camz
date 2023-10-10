// Copyright © 2022 Sloan Childers
package blackjack

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/blackjack/webcam"
	"github.com/osintami/camz/base"
	"github.com/rs/zerolog/log"
)

type Driver struct {
	config *base.CameraConfig
	webcam *webcam.Webcam
	frame  []byte
	mutex  sync.Mutex
	stop   bool
}

const (
	V4L2_PIX_FMT_YUYV = 0x56595559
	V4L2_PIX_FMT_MJPG = 0x47504A4D
)

func NewDriver(config *base.CameraConfig) *Driver {
	return &Driver{
		config: config,
		frame:  base.EmptyFrame(config.Width, config.Height),
	}
}

func (x *Driver) ListFormatsAndFrameSizes() base.Formats {
	formats := base.Formats{}
	format_desc := x.webcam.GetSupportedFormats()
	for formatObj, formatStr := range format_desc {
		format := base.Format{Name: formatStr}
		sizes := []base.Size{}
		fs := base.FrameSizes(x.webcam.GetSupportedFrameSizes(formatObj))
		sort.Sort(fs)
		for _, frameSize := range fs {
			sizes = append(sizes, base.Size{Size: frameSize.GetString()})
		}
		format.Sizes = sizes
		formats.Formats = append(formats.Formats, format)
	}
	return formats
}

func (x *Driver) Grab() base.IFrame {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	frame := base.NewFrame(x.config)
	frame.SetImage(x.frame, base.JPEG)
	return frame
}

func (x *Driver) grab() []byte {
	var out []byte
	err := x.webcam.WaitForFrame(1)
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Msg("WaitForFrame")
	} else {
		out, err = x.webcam.ReadFrame()
		if err != nil || len(out) == 0 {
			log.Warn().Str("component", "driver").Str("name", x.config.Name).Msg("ReadFrame")
			out = base.EmptyFrame(x.config.Width, x.config.Height)
		}
	}
	// NOTE:  must make a copy of the out slice
	return base.Copy(out)
}

func (x *Driver) Stream() {
	go x.stream()
}

func (x *Driver) CheckSize(width, height uint32) bool {
	format_desc := x.webcam.GetSupportedFormats()
	for format, _ := range format_desc {
		frameSizes := x.webcam.GetSupportedFrameSizes(format)
		for _, frameSize := range frameSizes {
			if frameSize.MaxHeight == height && frameSize.MaxWidth == width {
				return true
			}
		}
	}
	return false
}

func (x *Driver) Open() error {
	var err error
	x.webcam, err = webcam.Open(fmt.Sprintf("/dev/video%d", x.config.Device))
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Int("device", x.config.Device).Msg("webcam.Open")
		return nil
	}

	err = x.webcam.SetFramerate(float32(x.config.Rate))
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("SetFramerate")
		return err
	}

	err = x.webcam.SetAutoWhiteBalance(true)
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("SetAutoWhiteBalance")
		return err
	}

	//	switch x.config.Format {
	//	case "Motion-JPEG":
	_, _, _, err = x.webcam.SetImageFormat(V4L2_PIX_FMT_MJPG, uint32(x.config.Width), uint32(x.config.Height))
	//	case "YUYV 4:2:2":
	//		_, _, _, err = x.webcam.SetImageFormat(V4L2_PIX_FMT_YUYV, uint32(x.config.Width), uint32(x.config.Height))
	//	}
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("SetImageFormat")
		return err
	}
	x.stop = false
	return x.webcam.StartStreaming()
}

func (x *Driver) Stop() {
	x.mutex.Lock()
	x.stop = true
	x.frame = base.EmptyFrame(x.config.Width, x.config.Height)
	err := x.webcam.StopStreaming()
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("StopStreaming")
	}
	x.webcam.Close()
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
		x.frame = x.grab()
		x.mutex.Unlock()
		base.Sleep(int64(x.config.Rate), time.Now().UnixMilli()-startTime)
	}
}
