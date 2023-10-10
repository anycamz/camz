// Copyright © 2023 <Sloan Childers>
package base

import (
	"bufio"
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"
)

type Frame struct {
	name      string
	width     int
	height    int
	img       gocv.Mat
	jpeg      []byte
	frameTime time.Time
	mutex     sync.Mutex
}

func NewFrame(config *CameraConfig) IFrame {
	return &Frame{
		name:      config.Name,
		width:     config.Width,
		height:    config.Height,
		img:       gocv.NewMat(),
		frameTime: time.Now()}
}

func (x *Frame) Clone() IFrame {
	return &Frame{
		name:      x.name,
		width:     x.width,
		height:    x.height,
		img:       x.img.Clone(),
		frameTime: x.frameTime}
}

func (x *Frame) Time() time.Time {
	return x.frameTime
}

func (x *Frame) Close() {
	x.img.Close()
}

func (x *Frame) Width() int {
	return x.width
}

func (x *Frame) Height() int {
	return x.height
}

func (x *Frame) Empty() bool {
	return x.img.Empty()
}

func (x *Frame) SetImage(img interface{}, typeCode int) {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	var data gocv.Mat
	switch typeCode {
	case GOCV:
		data = img.(gocv.Mat)
	case GRAYSCALE8:
		data = gocv.NewMat()
		err := gocv.IMDecodeIntoMat(img.([]byte), gocv.IMReadGrayScale, &data)
		if err != nil {
			log.Error().Err(err).Str("component", "frame").Str("name", x.name).Msg("JPEG decode with OpenCV")
		}
	case JPEG:
		x.jpeg = img.([]byte)
		data = gocv.NewMat()
		err := gocv.IMDecodeIntoMat(img.([]byte), gocv.IMReadAnyColor, &data)
		if err != nil {
			log.Error().Err(err).Str("component", "frame").Str("name", x.name).Msg("JPEG decode with OpenCV")
		}
		if false {
			gocv.IMWrite("jpeg.jpg", data)
			os.Exit(0)
		}
	}
	x.frameTime = time.Now()
	x.img.Close()
	x.img = data
}

func (x *Frame) ToColorJpeg(exif *ExifInfo) []byte {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	var jpeg []byte
	decorate := true // mimic opencv motion detection and markup the images
	if x.jpeg == nil || decorate {
		data, err := gocv.IMEncodeWithParams(gocv.JPEGFileExt, x.img, []int{gocv.IMWriteJpegQuality, 50})
		if err != nil {
			log.Error().Err(err).Str("component", "camera").Msg("JPEG encode")
			return EmptyFrame(x.width, x.height)
		}
		defer data.Close()
		x.jpeg = Copy(data.GetBytes())
	}
	if exif != nil {
		var err error
		jpeg, err = WriteExif(exif, "sloanasan", "OSINTAMI", "Camz 1.0", "localhost", x.frameTime, x.jpeg)
		if err != nil {
			log.Error().Err(err).Str("component", "camera").Msg("JPEG encode")
			return x.jpeg
		}
		x.jpeg = jpeg
	}
	return x.jpeg
}

// func (x *Frame) ToGrayscaleJpeg(exif *ExifInfo) []byte {
// 	x.mutex.Lock()
// 	defer x.mutex.Unlock()
// 	return x.ToColorJpeg(exif)
// }

func (x *Frame) ToGrayscale() gocv.Mat {
	gray := gocv.NewMat()
	gocv.CvtColor(x.img, &gray, gocv.ColorBGRToGray)
	return gray
}

func (x *Frame) ToBytes() []byte {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	return x.img.ToBytes()
}

func (x *Frame) OpenCV(clone bool) gocv.Mat {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	if clone {
		return x.img.Clone()
	} else {
		return x.img
	}
}

func EmptyFrame(width, height int) []byte {
	pix := make([]uint8, width*height*4)
	// random static
	//rand.Read(pix)

	// black
	for i := 0; i < len(pix); i++ {
		pix[i] = 0x00
	}
	img := &image.NRGBA{
		Pix:    pix,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	jpeg.Encode(w, img, &jpeg.Options{Quality: 10})
	return b.Bytes()
}

func Copy(slice []byte) []byte {
	out := make([]byte, len(slice))
	copy(out, slice)
	return out
}
