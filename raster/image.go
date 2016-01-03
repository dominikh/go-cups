package raster

import (
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
