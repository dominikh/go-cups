package raster

import "image/color"

func (p *Page) ParseColors(b []byte) ([]color.Color, error) {
	switch p.Header.CUPSColorSpace {
	case ColorSpaceBlack:
		return p.parseColorsBlack(b), nil
	default:
		return nil, ErrUnsupported
	}
}

func (p *Page) parseColorsBlack(b []byte) []color.Color {
	// TODO don't assume chunked
	var colors []color.Color
	for _, packet := range b {
		for i := uint(0); i < 8; i++ {
			// TODO This doesn't depend on endianness in any way, does it?
			if packet<<i&128 == 0 {
				colors = append(colors, color.Gray{255})
			} else {
				colors = append(colors, color.Gray{0})
			}
		}
	}
	return colors
}
