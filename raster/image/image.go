package image

import (
	"image"
	"image/color"

	"honnef.co/go/cups/raster"
)

// FIXME respect bounding boxes

func rect(p *raster.Page) image.Rectangle {
	// TODO respect bounding box
	return image.Rect(0, 0, int(p.Header.CUPSWidth), int(p.Header.CUPSHeight))
}

// Image returns an image.Image of the page.
//
// Depending on the color space and bit depth used, image.Image
// implementations from this package or from the Go standard library
// image package may be used. The mapping is as follows:
//
//   - 1-bit, ColorSpaceBlack -> *Monochrome
//   - 8-bit, ColorSpaceBlack -> *image.Gray
//   - 8-bit, ColorSpaceCMYK -> *image.CMYK
//   - Other combinations are not currently supported and will return
//     ErrUnsupported. They might be added in the future.
//
// No calls to ReadLine or ReadAll must be made before or after
// calling Image. That is, Image consumes the entire stream of the
// page.
//
// Note that decoding an entire page at once may use considerable
// amounts of memory. For efficient, line-wise processing, a
// combination of ReadLine and ParseColors should be used instead.
func Image(p *raster.Page) (image.Image, error) {
	b := make([]byte, p.TotalSize())
	err := p.ReadAll(b)
	if err != nil {
		return nil, err
	}

	// FIXME support color orders other than chunked
	if p.Header.CUPSColorOrder != raster.ChunkyPixels {
		return nil, raster.ErrUnsupported
	}
	switch p.Header.CUPSColorSpace {
	case raster.ColorSpaceBlack:
		switch p.Header.CUPSBitsPerColor {
		case 1:
			return &Monochrome{
				Pix:    b,
				Stride: int(p.Header.CUPSBytesPerLine),
				Rect:   rect(p),
			}, nil
		case 8:
			for i, v := range b {
				b[i] = 255 - v
			}
			return &image.Gray{
				Pix:    b,
				Stride: int(p.Header.CUPSBytesPerLine),
				Rect:   rect(p),
			}, nil
		default:
			return nil, raster.ErrUnsupported
		}
	case raster.ColorSpaceCMYK:
		if p.Header.CUPSBitsPerColor != 8 {
			return nil, raster.ErrUnsupported
		}
		// TODO does cups have a byte order for colors in a pixel and
		// do we need to swap bytes?
		return &image.CMYK{
			Pix:    b,
			Stride: int(p.Header.CUPSBytesPerLine),
			Rect:   rect(p),
		}, nil
	default:
		return nil, raster.ErrUnsupported
	}
}

var _ image.Image = (*Monochrome)(nil)

// Monochrome is an in-memory monochromatic image, with 8 pixels
// packed into one byte. Its At method returns color.Gray values.
type Monochrome struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func (img *Monochrome) ColorModel() color.Model {
	return color.GrayModel
}

func (img *Monochrome) Bounds() image.Rectangle {
	return img.Rect
}

func (img *Monochrome) At(x, y int) color.Color {
	idx := img.PixOffset(x, y)
	if img.Pix[idx]<<uint(x%8)&128 == 0 {
		return color.Gray{Y: 255}
	}
	return color.Gray{Y: 0}
}

// PixOffset returns the index of the first element of Pix that
// corresponds to the pixel at (x, y).
func (img *Monochrome) PixOffset(x, y int) int {
	// TODO respect non-zero starting point of bounding box
	return y*img.Stride + (x / 8)
}
