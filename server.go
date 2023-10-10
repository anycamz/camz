// Copyright © 2023 Sloan Childers
package main

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/osintami/camz/base"
	"github.com/osintami/camz/sink"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"
)

type Config struct {
	PathPrefix string `env:"PATH_PREFIX" envDefault:"/"`
	ListenAddr string `env:"LISTEN_ADDR,required" envDefault:"0.0.0.0:80"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"TRACE"`
}

type CamzServer struct {
	webcam base.IDriver
	motion base.IMotion
	gps    *base.GPS
	config *base.CameraConfig
}

var ErrSizeUnsupported = errors.New("invalid size")
var ErrSaveConfig = errors.New("save configuration failed")
var ErrApiKey = errors.New("api key invalid")

func NewCamzServer(webcam base.IDriver, motion base.IMotion, gps *base.GPS, config *base.CameraConfig) *CamzServer {
	return &CamzServer{
		webcam: webcam,
		motion: motion,
		gps:    gps,
		config: config}
}

func (x *CamzServer) checkAPIKey(r *http.Request) bool {

	if true {
		return true
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		key = r.Header.Get("X-Api-Key")
	}

	if key == x.config.ApiKey {
		return true
	}

	return false
}

func (x *CamzServer) CommandHandler(w http.ResponseWriter, r *http.Request) {
	if !x.checkAPIKey(r) {
		sink.SendError(w, ErrApiKey, http.StatusForbidden)
		return
	}

	format := r.URL.Query().Get("command")
	switch format {
	case "stop":
		x.webcam.Stop()
	case "start":
		x.webcam.Open()
		x.webcam.Stream()
	case "reset":
		x.webcam.Reset()
	}
}

func (x *CamzServer) StreamHandler(w http.ResponseWriter, r *http.Request) {
	if !x.checkAPIKey(r) {
		sink.SendError(w, ErrApiKey, http.StatusForbidden)
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "h264":
		//x.StreamH264(w)
	case "wav":
		//x.StreamWAV(w)
	default:
		x.StreamMJPEG(w)
	}
}

var orange = color.RGBA{255, 127, 0, 0}

func (x *CamzServer) StreamMJPEG(w http.ResponseWriter) {

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=--myboundary")
	w.Header().Set("Server", "Camd")
	w.Header().Set("Connection", "Close")
	for {
		startTime := time.Now().UnixMilli()
		frame := x.webcam.Grab()
		intruder := false
		if x.config.Motion.Enabled {
			intruder = x.motion.Detect(frame)
			if intruder {
				if x.config.Motion.Decorate {
					currFrame := frame.OpenCV(false)
					gocv.Rectangle(&currFrame, image.Rect(0, 0, x.config.Width, x.config.Height), orange, 2)
				}
			}
		}
		jpeg := frame.ToColorJpeg(nil)
		if x.config.Motion.Enabled && intruder {
			exifInfo, err := x.gps.ToExif()
			if err == nil {
				// TODO:  these values needs to live in camera.json
				out, err := base.WriteExif(exifInfo, "OSINTAMI", "CarCamz", "0.1", "camz-dev", frame.Time(), jpeg)
				if err == nil {
					jpeg = out
					//os.WriteFile("gps.jpeg", jpeg, 0644)
				}
			}
		}
		frame.Close()

		if !base.ValidateJPEG(jpeg) {
			log.Warn().Str("component", "mjpeg-server").Str("name", x.config.Name).Msg("invalid JPEG, skipping")
			jpeg = base.EmptyFrame(x.config.Width, x.config.Height)
		}
		err := base.WriteMjpeg(w, jpeg)
		if err != nil {
			log.Warn().Str("component", "mjpeg-server").Str("name", x.config.Name).Msg("stream is dead")
			return
		}
		base.Sleep(int64(x.config.Rate), time.Now().UnixMilli()-startTime)
	}
}

func (x *CamzServer) FormatsHandler(w http.ResponseWriter, r *http.Request) {
	if !x.checkAPIKey(r) {
		sink.SendError(w, ErrApiKey, http.StatusForbidden)
		return
	}
	x.webcam.Stop()
	out, _ := json.Marshal(x.webcam.ListFormatsAndFrameSizes())
	x.webcam.Open()
	x.webcam.Stream()
	w.Write(out)

}

func (x *CamzServer) ConfigUpdateHandler(w http.ResponseWriter, r *http.Request) {

	if !x.checkAPIKey(r) {
		sink.SendError(w, ErrApiKey, http.StatusForbidden)
		return
	}

	config := &base.CameraConfig{Motion: &base.MotionConfig{}}

	// establish default values for missing fields
	config.ApiKey = x.config.ApiKey
	config.Name = x.config.Name
	config.Device = x.config.Device
	config.Height = x.config.Height
	config.Width = x.config.Width
	config.Rate = x.config.Rate
	config.Motion.Enabled = x.config.Motion.Enabled
	config.Motion.Area = x.config.Motion.Area
	config.Motion.Detections = x.config.Motion.Detections
	config.Motion.Overlap = x.config.Motion.Overlap
	config.Motion.Decorate = x.config.Motion.Decorate

	err := json.NewDecoder(r.Body).Decode(config)
	if err != nil {
		sink.SendError(w, err, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// validate width/height against supported sizes
	// if !x.webcam.CheckSize(uint32(config.Width), uint32(config.Height)) {
	// 	sink.SendError(w, ErrSizeUnsupported, http.StatusInternalServerError)
	// 	return
	// }

	// backup current settings
	backup := &base.CameraConfig{Motion: &base.MotionConfig{}}
	backup.ApiKey = x.config.ApiKey
	backup.Name = x.config.Name
	backup.Device = x.config.Device
	backup.Height = x.config.Height
	backup.Width = x.config.Width
	backup.Rate = x.config.Rate
	backup.Motion.Enabled = x.config.Motion.Enabled
	backup.Motion.Area = x.config.Motion.Area
	backup.Motion.Detections = x.config.Motion.Detections
	backup.Motion.Overlap = x.config.Motion.Overlap
	backup.Motion.Decorate = x.config.Motion.Decorate

	// merge with live settings
	x.config.ApiKey = config.ApiKey
	x.config.Name = config.Name
	x.config.Device = config.Device
	x.config.Height = config.Height
	x.config.Width = config.Width
	x.config.Rate = config.Rate
	x.config.Motion.Enabled = config.Motion.Enabled
	x.config.Motion.Area = config.Motion.Area
	x.config.Motion.Detections = config.Motion.Detections
	x.config.Motion.Overlap = config.Motion.Overlap
	x.config.Motion.Decorate = config.Motion.Decorate

	err = x.webcam.Reset()
	if err != nil {
		// revert settings
		x.config.ApiKey = backup.ApiKey
		x.config.Name = backup.Name
		x.config.Device = backup.Device
		x.config.Height = backup.Height
		x.config.Width = backup.Width
		x.config.Rate = backup.Rate
		x.config.Motion.Enabled = backup.Motion.Enabled
		x.config.Motion.Area = backup.Motion.Area
		x.config.Motion.Detections = backup.Motion.Detections
		x.config.Motion.Overlap = backup.Motion.Overlap
		x.config.Motion.Decorate = backup.Motion.Decorate
		x.webcam.Reset()
		sink.SendError(w, err, http.StatusInternalServerError)
	}

	// save new settings for next restart
	data, err := json.MarshalIndent(x.config, "", "   ")
	if err != nil {
		sink.SendError(w, ErrSaveConfig, http.StatusInternalServerError)
	}
	os.WriteFile("./camera.json", data, fs.ModeAppend)

	sink.SendPrettyJSON(r.Context(), w, x.config)
}

func (x *CamzServer) ConfigReadHandler(w http.ResponseWriter, r *http.Request) {
	if !x.checkAPIKey(r) {
		sink.SendError(w, ErrApiKey, http.StatusForbidden)
		return
	}

	sink.SendPrettyJSON(r.Context(), w, x.config)
}
