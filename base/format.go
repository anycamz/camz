// Copyright © 2015 <Oleksandr Senkovych>
// Copyright © 2023 <Sloan Childers>
package base

import (
	"github.com/blackjack/webcam"
)

type Size struct {
	Size string
}
type Format struct {
	Name  string
	Sizes []Size
}
type Formats struct {
	Formats []Format
}

type FrameSizes []webcam.FrameSize

func (slice FrameSizes) Len() int {
	return len(slice)
}

// For sorting purposes
func (slice FrameSizes) Less(i, j int) bool {
	ls := slice[i].MaxWidth * slice[i].MaxHeight
	rs := slice[j].MaxWidth * slice[j].MaxHeight
	return ls < rs
}

// For sorting purposes
func (slice FrameSizes) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
