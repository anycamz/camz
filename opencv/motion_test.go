// Copyright © 2023 Sloan Childers
package opencv

import (
	"image"
	"testing"

	"github.com/osintami/camz/base"
	"gocv.io/x/gocv"
)

func TestOverlaps(t *testing.T) {
	mc := &base.MotionConfig{Decorate: true}
	config := &base.CameraConfig{Motion: mc}

	x := &Motion{
		config:    config,
		lastFrame: gocv.NewMat()}

	img := gocv.NewMat()

	contours := []image.Rectangle{
		{image.Point{10, 10}, image.Point{100, 100}},
		{image.Point{20, 20}, image.Point{200, 200}},
		{image.Point{30, 30}, image.Point{300, 300}},
	}

	numOverlaps := x.Overlaps(img, contours)
	if numOverlaps != 3 {
		t.Errorf("Expected 3 overlapping contours, got %d", numOverlaps)
	}
}

func _TestDetect(t *testing.T) {
	mc := &base.MotionConfig{
		Enabled:       true,
		Area:          250,
		Detections:    3,
		Overlap:       0,
		Mask:          []base.MotionRectangle{},
		BeforeSeconds: 0,
		AfterSeconds:  0,
		Decorate:      true,
	}

	config := &base.CameraConfig{
		Motion: mc,
		Width:  320,
		Height: 240}

	// Create a Motion object
	motion := &Motion{config: config}

	// Create a GoCV Mat object
	img := gocv.NewMat()
	frame := base.NewFrame(config)
	frame.SetImage(img, base.GOCV)

	// Call the Detect function

	motionDetected := motion.Detect(frame)

	// Assert that motion was detected
	if !motionDetected {
		t.Errorf("Expected motion to be detected, but it was not")
	}
}
