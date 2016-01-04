package raster

import (
	"image"
	"image/color"
	"io"
)

func (p *Page) ParseColors(b []byte) ([]color.Color, error) {
	// TODO support banded and planar
	if p.Header.CUPSColorOrder != ChunkyPixels {
		return nil, ErrUnsupported
	}
	switch p.Header.CUPSColorSpace {
	case ColorSpaceBlack:
		return p.parseColorsBlack(b), nil
	default:
		return nil, ErrUnsupported
	}
}

func (p *Page) parseColorsBlack(b []byte) []color.Color {
	var colors []color.Color
	for _, packet := range b {
		for i := uint(0); i < 8; i++ {
			if packet<<i&128 == 0 {
				colors = append(colors, color.Gray{255})
			} else {
				colors = append(colors, color.Gray{0})
			}
		}
	}
	return colors
}

type ImageSetter interface {
	Set(x, y int, c color.Color)
}

// Render renders a CUPS raster image onto any image.Image that
// implements the Set method.
func (p *Page) Render(img ImageSetter) error {
	b := make([]byte, p.LineSize())
	for y := uint32(0); y < p.Header.CUPSHeight; y++ {
		err := p.ReadLine(b)
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		if err != nil {
			return err
		}
		colors, err := p.ParseColors(b)
		if err != nil {
			return err
		}
		for x, color := range colors {
			img.Set(x, int(y), color)
		}
	}
	return nil
}

func (p *Page) Image() image.Image {
	b := make([]byte, p.TotalSize())
	_ = p.ReadAll(b)

	// FIXME support color orders other than chunked
	switch p.Header.CUPSColorSpace {
	case ColorSpaceBlack:
		return &Monochrome{p: p, data: b}
	}
}

var _ image.Image = (*Monochrome)(nil)

type Monochrome struct {
	p    *Page
	data []byte
}

func (img *Monochrome) ColorModel() color.Model {
	return color.GrayModel
}

func (img *Monochrome) Bounds() image.Rectangle {
	return image.Rect(0, 0, int(img.p.Header.CUPSWidth), int(img.p.Header.CUPSHeight))
}

func (img *Monochrome) At(x, y int) color.Color {
	idx := y*int(img.p.Header.CUPSBytesPerLine) + (x / 8)
	if img.data[idx]<<uint(x%8)&128 > 0 {
		return color.Gray{Y: 0}
	}
	return color.Gray{Y: 255}
}