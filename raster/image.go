package raster

import (
	"image"
	"image/color"
)

// FIXME respect bounding boxes

// ParseColors parses b and returns the colors stored in it, one per
// pixel.
//
// It currently supports the following color spaces and bit depths,
// although more might be added later:
//
// - 1-bit, ColorSpaceBlack -> color.Gray
// - 8-bit, ColorSpaceCMYK -> color.CMYK
func (p *Page) ParseColors(b []byte) ([]color.Color, error) {
	// TODO support banded and planar
	if p.Header.CUPSColorOrder != ChunkyPixels {
		return nil, ErrUnsupported
	}
	switch p.Header.CUPSColorSpace {
	case ColorSpaceBlack:
		return p.parseColorsBlack(b)
	case ColorSpaceCMYK:
		return p.parseColorsCMYK(b)
	default:
		return nil, ErrUnsupported
	}
}

func (p *Page) parseColorsBlack(b []byte) ([]color.Color, error) {
	if p.Header.CUPSBitsPerColor != 1 {
		// TODO support all depths
		return nil, ErrUnsupported
	}
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
	return colors, nil
}

func (p *Page) parseColorsCMYK(b []byte) ([]color.Color, error) {
	if p.Header.CUPSBitsPerColor != 8 {
		return nil, ErrUnsupported
	}
	if len(b)%4 != 0 || len(b) < 4 {
		return nil, ErrInvalidFormat
	}
	var colors []color.Color
	for i := 0; i < len(b); i += 4 {
		// TODO does cups have a byte order for colors in a pixel and
		// do we need to swap bytes?
		c := color.CMYK{C: b[i], M: b[i+1], Y: b[i+2], K: b[i+3]}
		colors = append(colors, c)
	}
	return colors, nil
}

func (p *Page) rect() image.Rectangle {
	return image.Rect(0, 0, int(p.Header.CUPSWidth), int(p.Header.CUPSHeight))
}

// Image returns an image.Image of the page.
//
// Depending on the color space and bit depth used, image.Image
// implementations from this package or from the Go standard library
// image package may be used. The mapping is as follows:
//
// - 1-bit, ColorSpaceBlack -> *Monochrome
// - 8-bit, ColorSpaceCMYK -> *image.CMYK
// - Other combinations are not currently supported and will return
//   ErrUnsupported. They might be added in the future.
//
// No calls to ReadLine or ReadAll must be made before or after
// calling Image. That is, Image consumes the entire stream of the
// current page.
//
// Note that decoding an entire page at once may use considerable
// amounts of memory. For efficient, line-wise processing, a
// combination of ReadLine and ParseColors should be used instead.
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
