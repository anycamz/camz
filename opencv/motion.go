// Copyright © 2023 Sloan Childers
package opencv

import (
	"image"
	"image/color"

	"github.com/osintami/camz/base"
	"gocv.io/x/gocv"
)

var green = color.RGBA{0, 255, 0, 0}
var yellow = color.RGBA{255, 255, 0, 0}

type Motion struct {
	config    *base.CameraConfig
	lastFrame gocv.Mat
}

func NewMotion(config *base.CameraConfig) base.IMotion {
	return &Motion{
		config:    config,
		lastFrame: gocv.NewMat()}
}

func (x *Motion) Overlaps(currFrame gocv.Mat, contours []image.Rectangle) int {
	num := 0
	for i := 0; i < len(contours); i++ {
		for j := i + 1; j < len(contours); j++ {
			if !contours[i].Intersect(contours[j]).Empty() {
				if x.config.Motion.Decorate {
					gocv.Rectangle(&currFrame, contours[i], yellow, 1)
					gocv.Rectangle(&currFrame, contours[j], yellow, 1)
				}
				num++
			}
		}
	}
	return num
}

func (x *Motion) Detect(in base.IFrame) bool {
	currFrame := in.OpenCV(false)
	//SaveToFile("current.jpeg", currFrame)
	edges := x.prepareFrame(currFrame)

	if x.lastFrame.Empty() {
		x.lastFrame.Close()
		x.lastFrame = edges
		return false
	}

	diffFrame := gocv.NewMat()
	defer diffFrame.Close()
	gocv.AbsDiff(x.lastFrame, edges, &diffFrame)
	//SaveToFile("edges.jpeg", diffFrame)
	x.lastFrame.Close()
	x.lastFrame = edges

	contours := gocv.FindContours(diffFrame, gocv.RetrievalTree, gocv.ChainApproxSimple)
	defer contours.Close()

	rects := []image.Rectangle{}

	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)
		if gocv.ContourArea(contour) < x.config.Motion.Area {
			continue
		}
		rects = append(rects, gocv.BoundingRect(contour))
		if x.config.Motion.Decorate {
			gocv.Rectangle(&currFrame, gocv.BoundingRect(contour), green, 1)
		}
	}

	if x.config.Motion.Decorate {
		color := gocv.NewMat()
		gocv.CvtColor(edges, &color, gocv.ColorGrayToBGR)
		blended := gocv.NewMat()
		gocv.AddWeighted(currFrame, 0.8, color, 0.4, 0, &blended)
		color.Close()
		blended.CopyTo(&currFrame)
		blended.Close()
		//base.SaveToFile("motion.jpeg", currFrame)
	}

	return x.Overlaps(currFrame, rects) > x.config.Motion.Overlap
}

func (x *Motion) prepareFrame(currFrame gocv.Mat) gocv.Mat {

	// 10-15% more CPU, but better in low light and overall
	canny := gocv.NewMat()
	gocv.Canny(currFrame, &canny, 50, 100)

	// mask areas to ignore in white
	for _, mask := range x.config.Motion.Mask {
		gocv.Rectangle(&canny, image.Rectangle{image.Point{mask.Px1, mask.Py1}, image.Point{mask.Px2, mask.Py2}}, color.RGBA{255, 255, 255, 0}, -1)
	}
	return canny
}

// func SaveToFile(file string, frame gocv.Mat) {
// 	if false {
// 		gocv.IMWrite(file, frame)
// 	}
// }
