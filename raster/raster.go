package raster

import "image/color"

const (
	AdvanceNever     = 0
	AdvanceAfterFile = 1
	AdvanceAfterJob  = 2
	AdvanceAfterSet  = 3
	AdvanceAfterPage = 4
)

const (
	CutNever     = 0
	CutAfterFile = 1
	CutAfterJob  = 2
	CutAfterSet  = 3
	CutAfterPage = 4
)

const (
	JogNever     = 0
	JogAfterFile = 1
	JogAfterJob  = 2
	JogAfterSet  = 3
)

const (
	EdgeTop    = 0
	EdgeRight  = 1
	EdgeBottom = 2
	EdgeLeft   = 3
)

const (
	RotateNone             = 0
	RotateCounterClockwise = 1
	RotateUpsideDown       = 2
	RotateClockwise        = 3
)

const (
	ChunkyPixels = 0
	BandedPixels = 1
	PlanarPixels = 2
)

const (
	ColorSpaceGray     = 0
	ColorSpaceRGB      = 1
	ColorSpaceRGBA     = 2
	ColorSpaceBlack    = 3
	ColorSpaceCMY      = 4
	ColorSpaceYMC      = 5
	ColorSpaceCMYK     = 6
	ColorSpaceYMCK     = 7
	ColorSpaceKCMY     = 8
	ColorSpaceKCMYcm   = 9
	ColorSpaceGMCK     = 10
	ColorSpaceGMCS     = 11
	ColorSpaceWHITE    = 12
	ColorSpaceGOLD     = 13
	ColorSpaceSILVER   = 14
	ColorSpaceCIEXYZ   = 15
	ColorSpaceCIELab   = 16
	ColorSpaceRGBW     = 17
	ColorSpacesGray    = 18
	ColorSpacesRGB     = 19
	ColorSpaceAdobeRGB = 20
	ColorSpaceICC1     = 32
	ColorSpaceICC2     = 33
	ColorSpaceICC3     = 34
	ColorSpaceICC4     = 35
	ColorSpaceICC5     = 36
	ColorSpaceICC6     = 37
	ColorSpaceICC7     = 38
	ColorSpaceICC8     = 39
	ColorSpaceICC9     = 40
	ColorSpaceICCA     = 41
	ColorSpaceICCB     = 42
	ColorSpaceICCC     = 43
	ColorSpaceICCD     = 44
	ColorSpaceICCE     = 45
	ColorSpaceICCF     = 46
	ColorSpaceDevice1  = 48
	ColorSpaceDevice2  = 49
	ColorSpaceDevice3  = 50
	ColorSpaceDevice4  = 51
	ColorSpaceDevice5  = 52
	ColorSpaceDevice6  = 53
	ColorSpaceDevice7  = 54
	ColorSpaceDevice8  = 55
	ColorSpaceDevice9  = 56
	ColorSpaceDeviceA  = 57
	ColorSpaceDeviceB  = 58
	ColorSpaceDeviceC  = 59
	ColorSpaceDeviceD  = 60
	ColorSpaceDeviceE  = 61
	ColorSpaceDeviceF  = 62
)

type BoundingBox struct {
	Left   uint32
	Bottom uint32
	Right  uint32
	Top    uint32
}

type CUPSBoundingBox struct {
	Left   float32
	Bottom float32
	Right  float32
	Top    float32
}

type Header struct {
	// v1

	MediaClass       string
	MediaColor       string
	MediaType        string
	OutputType       string
	AdvanceDistance  uint32
	AdvanceMedia     int
	Collate          bool
	CutMedia         int
	Duplex           bool
	HorizDPI         uint32
	VertDPI          uint32
	BoundingBox      BoundingBox
	InsertSheet      bool
	Jog              int
	LeadingEdge      int
	MarginLeft       uint32
	MarginBottom     uint32
	ManualFeed       bool
	MediaPosition    uint32
	MediaWeight      uint32
	MirrorPrint      bool
	NegativePrint    bool
	NumCopies        uint32
	Orientation      int
	OutputFaceUp     bool
	Width            uint32
	Length           uint32
	Separations      bool
	TraySwitch       bool
	Tumble           bool
	CUPSWidth        uint32
	CUPSHeight       uint32
	CUPSMediaType    uint32
	CUPSBitsPerColor uint32
	CUPSBitsPerPixel uint32
	CUPSBytesPerLine uint32
	CUPSColorOrder   int
	CUPSColorSpace   int
	CUPSCompression  uint32
	CUPSRowCount     uint32
	CUPSRowFeed      uint32
	CUPSRowStep      uint32

	// v2, v3
	CUPSNumColors               uint32
	CUPSBorderlessScalingFactor float32
	CUPSPageSize                [2]float32
	CUPSImagingBBox             CUPSBoundingBox
	CUPSInteger                 [16]uint32
	CUPSReal                    [16]float32
	CUPSString                  [16]string
	CUPSMarkerType              string
	CUPSRenderingIntent         string
	CUPSPageSizeName            string
}

// ParseColors parses b and returns the colors stored in it, one per
// pixel.
//
// It currently supports the following color spaces and bit depths,
// although more might be added later:
//
//   - 1-bit, ColorSpaceBlack -> color.Gray
//   - 8-bit, ColorSpaceBlack -> color.Gray
//   - 8-bit, ColorSpaceCMYK -> color.CMYK
//
// Note that b might contain data for more colors than are actually
// present. This happens when data is stored with less than 8 bits per
// pixel. A page with 633 pixels per line would necessarily contain
// data for 640 pixels, as pixels 633-640 are stored in the same byte.
// When parsing ReadLine data, make sure to truncate the returned
// slice to the length of a single line. When parsing ReadAll data,
// the stride with which the resulting slice of colors is accessed has
// to be rounded up. Alternatively, ReadLineColors and ReadAllColors
// may be used, which return slices of colors and truncate them as
// needed.
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
	// TODO support all depths
	var colors []color.Color
	switch p.Header.CUPSBitsPerColor {
	case 1:
		for _, packet := range b {
			for i := uint(0); i < 8; i++ {
				if packet<<i&128 == 0 {
					colors = append(colors, color.Gray{255})
				} else {
					colors = append(colors, color.Gray{0})
				}
			}
		}
	case 8:
		for _, v := range b {
			colors = append(colors, color.Gray{Y: 255 - v})
		}
	default:
		return nil, ErrUnsupported
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

// LineSize returns the size of a single line, in bytes.
func (p *Page) LineSize() uint32 {
	return p.Header.CUPSBytesPerLine
}

// TotalSize returns the size of the entire page, in bytes.
func (p *Page) TotalSize() uint32 {
	return p.Header.CUPSHeight * p.Header.CUPSBytesPerLine
}
