// Copyright © 2023 Sloan Childers
package base

import (
	"io"
	"time"

	"gocv.io/x/gocv"
)

type ServerConfig struct {
	Version   string
	Name      string
	Uuid      string
	ClipHost  string
	ClipKey   string
	EmailTo   string
	EmailFrom string
	EmailHost string
	EmailPort int
	EmailPass string
}

type Cameras struct {
	Cameras []*CameraConfig
}

type CameraConfig struct {
	Enabled bool
	Uuid    string
	Device  int
	Name    string
	Width   int
	Height  int
	Rate    float32
	Motion  *MotionConfig
	Plugin  string
	// for network cameras
	Addr   string `json:"Addr,omitempty"`
	Port   int    `json:"Port,omitempty"`
	Uri    string `json:"Uri,omitempty"`
	User   string `json:"User,omitempty"`
	Pass   string `json:"Pass,omitempty"`
	ApiKey string `json:"ApiKey,omitempty"`
}

type MotionRectangle struct {
	Px1 int
	Py1 int
	Px2 int
	Py2 int
}

type MotionConfig struct {
	Enabled       bool
	Area          float64
	Detections    int
	Overlap       int
	Mask          []MotionRectangle
	BeforeSeconds int
	AfterSeconds  int
	Decorate      bool
}

type ICamera interface {
	Name() string
	Open() error
	Start()
	Stop()
	Reset()
	Grab() IFrame
	Stream(int, int, int, io.Writer)
}

type IDriver interface {
	Open() error
	Stop()
	Reset() error
	Stream()
	Grab() IFrame
	ListFormatsAndFrameSizes() Formats
}

type IFrame interface {
	Width() int
	Height() int
	Empty() bool
	Close()
	ToColorJpeg(*ExifInfo) []byte
	ToGrayscale() gocv.Mat
	//	ToGrayscaleJpeg(*ExifInfo) []byte
	OpenCV(bool) gocv.Mat
	ToBytes() []byte
	SetImage(interface{}, int) // []byte or gocv.Mat
	Clone() IFrame
	Time() time.Time
}

type IMotion interface {
	Detect(img IFrame) bool
}
