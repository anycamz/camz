package base

import (
	"bytes"
	"testing"
)

func TestEmptyFrame(t *testing.T) {
	// Create a new empty frame.
	jpeg := EmptyFrame(100, 100)

	// Check that the frame has the correct width and height.
	if len(jpeg) != 788 {
		t.Errorf("Expected frame to be 788 bytes, got %d bytes", len(jpeg))
	}
}

func TestCopy(t *testing.T) {
	// Create a new slice.
	slice := []byte{1, 2, 3, 4, 5}

	// Copy the slice.
	copy := Copy(slice)

	// Check that the copy is equal to the original slice.
	if !bytes.Equal(slice, copy) {
		t.Errorf("Expected copy to be equal to original slice, but it is not")
	}
}

func TestFrame_SetImage(t *testing.T) {
	// Create a new frame.
	frame := &Frame{}

	// Set the image data for the frame.
	jpeg := EmptyFrame(100, 100)
	frame.SetImage(jpeg, JPEG)

	// Check that the image data was set correctly.

	frameIn := frame.ToBytes()
	frameOut := frame.img.ToBytes()

	if bytes.Compare(frameIn, frameOut) != 0 {
		t.Errorf("Image data was not set correctly")
	}
}
