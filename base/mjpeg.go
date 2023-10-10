// Copyright © 2023 Sloan Childers
package base

import (
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"
)

func Sleep(frameRate int64, elapsedTime int64) {
	sleepTime := int64(1000/frameRate) - elapsedTime
	if sleepTime < 0 {
		return
	}
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
}

const (
	BOUNDARY          = "--myboundary"
	BOUNDARY_SIZE     = len(BOUNDARY)
	CONTENT_TYPE      = "Content-Type: image/jpeg"
	CONTENT_TYPE_SIZE = len(CONTENT_TYPE)
	// FRAME_TIME          = "Sleep-Time: "
	// FRAME_TIME_SIZE     = len(FRAME_TIME)
	CONTENT_LENGTH      = "Content-Length: "
	CONTENT_LENGTH_SIZE = len(CONTENT_LENGTH)
	EOL_SIZE            = 2
)

var EOL = []byte{'\r', '\n'}

func StreamMJPEG(x ICamera, width, height, rate int, out io.Writer) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Str("component", "webcam").Str("name", x.Name()).Msg("webcam stream crash")
			x.Reset()
		}
	}()

	for {
		startTime := time.Now().UnixMilli()
		err := WriteMjpeg(out, x.Grab().ToColorJpeg(nil))
		if err != nil {
			log.Warn().Str("component", "camera").Str("name", x.Name()).Msg("stream is dead")
			return
		}
		Sleep(int64(rate), time.Now().UnixMilli()-startTime)
	}
}

func WriteMjpeg(writer io.Writer, data []byte) error {
	writer.Write([]byte(BOUNDARY))
	writer.Write(EOL)
	writer.Write([]byte(CONTENT_TYPE))
	writer.Write(EOL)
	writer.Write([]byte(fmt.Sprintf("%s%d", CONTENT_LENGTH, len(data))))
	writer.Write(EOL)
	writer.Write(EOL)
	writer.Write(data)
	_, err := writer.Write(EOL)
	return err
}

const (
	JPEG_MARKER byte = 0xFF
	JPEG_SOI    byte = 0xD8
	JPEG_EOI    byte = 0xD9
)

func ValidateJPEG(data []byte) bool {
	size := len(data)
	if (data[0] == JPEG_MARKER) && (data[1] == JPEG_SOI) && (data[size-2] == JPEG_MARKER) && (data[size-1] == JPEG_EOI) {
		return true
	}
	return false
}

const (
	GRAYSCALE8 = 0
	JPEG       = 1
	RGB24      = 2
	BMP        = 3
	YUV422     = 4
	GOCV       = 5
)
