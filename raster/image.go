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

func (p *Page) rect() image.Rectangle {
	return image.Rect(0, 0, int(p.Header.CUPSWidth), int(p.Header.CUPSHeight))
}

func (p *Page) Image() (image.Image, error) {
	b := make([]byte, p.TotalSize())
	err := p.ReadAll(b)
	if err != nil {
		return nil, err
	}

	// FIXME support color orders other than chunked
	if p.Header.CUPSColorOrder != ChunkyPixels {
		return nil, ErrUnsupported
	}
	switch p.Header.CUPSColorSpace {
	case ColorSpaceBlack:
		return &Monochrome{p: p, data: b}, nil
	case ColorSpaceCMYK:
		if p.Header.CUPSBitsPerColor != 8 {
			return nil, ErrUnsupported
		}
		// TODO does cups have a byte order for colors in a pixel and
		// do we need to swap bytes?
		return &image.CMYK{
			Pix:    b,
			Stride: int(p.Header.CUPSBytesPerLine),
			Rect:   p.rect(),
		}, nil
	default:
		return nil, ErrUnsupported
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
	return img.p.rect()
}

func (img *Monochrome) At(x, y int) color.Color {
	idx := y*int(img.p.Header.CUPSBytesPerLine) + (x / 8)
	if img.data[idx]<<uint(x%8)&128 == 0 {
		return color.Gray{Y: 255}
	}
	return color.Gray{Y: 0}
}
