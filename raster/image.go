package raster

import (
	"image"
	"image/color"
	"io"
	"io/ioutil"
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

// TODO Instead of having a Render function, consider implementing an
// image.Image based on the RLE byte data.

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
	// FIXME this function is wrong in all ways imaginable
	b, err := ioutil.ReadAll(p.dec.r)
	if err != nil {
		panic(err)
	}
	return &Image{p: p, data: b}
}

var _ image.Image = (*Image)(nil)

type Image struct {
	p    *Page
	data []byte
	// y -> (x -> offset)
	ys map[int]int
}

func (img *Image) populateCache() {
	img.ys = map[int]int{}
	// TODO support other color spaces
	y := 0
	off := 0
	// FIXME deal with error
	bpc, err := bytesPerColor(img.p.Header)
	_ = err
	for off < len(img.data) {
		rep := int(img.data[off]) + 1
		for i := 0; i < rep; i++ {
			img.ys[y+i] = off + 1
		}
		y += rep
		off++

		x := 0
		for uint32(x) < img.p.Header.CUPSWidth {
			rep := int(img.data[off])
			off++
			if rep <= 127 {
				// rep repeating colors
				rep++
				off += bpc
			} else {
				// rep non-repeating colors
				rep = 257 - rep
				off += rep * bpc
			}
			pixels := rep * ((bpc * 8) / int(img.p.Header.CUPSBitsPerPixel))
			x += pixels
		}
	}
}

func (img *Image) ColorModel() color.Model {
	switch img.p.Header.CUPSColorSpace {
	case ColorSpaceBlack:
		return color.Palette{color.Gray{Y: 0}, color.Gray{Y: 255}}
	default:
		// TODO panic?
		return nil
	}
}

func (img *Image) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{},
		Max: image.Point{
			X: int(img.p.Header.CUPSWidth),
			Y: int(img.p.Header.CUPSHeight),
		},
	}
}

func (img *Image) At(x, y int) color.Color {
	if img.p.Header.CUPSColorSpace != ColorSpaceBlack {
		// TODO support other color spaces
		return nil
	}
	if int64(x) >= int64(img.p.Header.CUPSWidth) ||
		int64(y) >= int64(img.p.Header.CUPSHeight) {
		return nil
	}
	if img.ys == nil {
		img.populateCache()
	}

	off, ok := img.ys[y]
	if !ok {
		panic("invalid RLE cache")
	}

	// FIXME don't assume monochrome or chunked
	ppc := 8
	d := x / ppc
	colors := 0
	// FIXME deal with error
	bpc, err := bytesPerColor(img.p.Header)
	_ = err
	for {
		rep := int(img.data[off])
		off++

		b := -1
		if rep <= 127 {
			rep++
			if d >= colors && d <= colors+rep {
				b = int(img.data[off])
			}
			off += bpc
		} else {
			rep = 257 - rep
			if d >= colors && d <= colors+rep {
				b = int(img.data[off+(d-colors)*bpc])
			}
			off += rep * bpc
		}
		colors += rep
		if b > -1 {
			b := byte(b)
			// FIXME don't assume monochrome or chunked
			if b<<uint(x%8)&128 == 0 {
				return color.Gray{Y: 255}
			}
			return color.Gray{Y: 0}
		}
	}
}
