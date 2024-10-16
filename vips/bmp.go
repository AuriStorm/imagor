package vips

import (
	"image"
	"image/draw"
	"io"

	"golang.org/x/image/bmp"
)

func loadImageFromBMP(r io.Reader) (*Image, error) {
	img, err := bmp.Decode(r)
	if err != nil {
		return nil, err
	}
	rect := img.Bounds()
	size := rect.Size()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(rect)
		draw.Draw(rgba, rect, img, rect.Min, draw.Src)
	}
	return LoadImageFromMemory(rgba.Pix, size.X, size.Y, 4)
}
