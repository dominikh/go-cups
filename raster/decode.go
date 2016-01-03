package raster

// TODO provide two APIs for decoding:
// 1) decode the whole image into an image.Image (or a []byte?)
// 2) read one line at a time

// TODO instead of storing a line buffer in the Page, allow the user
// to pass a []byte to ReadLine. Add a method to Page that returns the
// correct size.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

var ErrUnsupported = errors.New("unsupported file format")
var ErrUnknownColorOrder = errors.New("unknown color order")

const (
	syncV1BE = "RaSt"
	syncV1LE = "tSaR"

	syncV2BE = "RaS2"
	syncV2LE = "2SaR"

	syncV3BE = "RaS3"
	syncV3LE = "3SaR"
)

func parseMagic(b []byte) (version int, bo binary.ByteOrder, ok bool) {
	if len(b) != 4 {
		return 0, nil, false
	}
	switch string(b) {
	case syncV1BE:
		return 1, binary.BigEndian, true
	case syncV2BE:
		return 2, binary.BigEndian, true
	case syncV3BE:
		return 3, binary.BigEndian, true
	case syncV1LE:
		return 1, binary.LittleEndian, true
	case syncV2LE:
		return 2, binary.LittleEndian, true
	case syncV3LE:
		return 3, binary.LittleEndian, true
	default:
		return 0, nil, false
	}
}

type Decoder struct {
	r       io.Reader
	bo      binary.ByteOrder
	err     error
	version int
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	d := &Decoder{r: r}
	magic := make([]byte, 4)
	_, err := io.ReadFull(r, magic)
	if err != nil {
		return nil, err
	}
	var ok bool
	d.version, d.bo, ok = parseMagic(magic)
	if !ok {
		return nil, ErrUnsupported
	}
	return d, nil
}

type Page struct {
	Header  *PageHeader
	dec     *Decoder
	line    []byte
	color   []byte
	lineRep int
}

// NextPage returns the next page in the raster stream. The returned
// page is only valid until the next call to NextPage.
func (d *Decoder) NextPage() (*Page, error) {
	// TODO if the user didn't read all lines, skip over them
	var err error
	var h *PageHeader
	switch d.version {
	case 1:
		h, err = d.decodeV1Header()
	case 2, 3:
		h, err = d.decodeV2Header()
	default:
		return nil, ErrUnsupported
	}
	if err != nil {
		return nil, err
	}
	bpc, err := bytesPerColor(h)
	if err != nil {
		return nil, err
	}
	p := &Page{
		Header: h,
		dec:    d,
		line:   make([]byte, 0, h.CUPSBytesPerLine),
		color:  make([]byte, bpc),
	}
	return p, nil
}

// ReadLine returns the next line of pixels in the image. The returned
// slice will only be valid until the next call to ReadLine.
func (p *Page) ReadLine() ([]byte, error) {
	if p.lineRep > 0 {
		p.lineRep--
		return p.line, nil
	}

	var lineRep byte
	err := binary.Read(p.dec.r, p.dec.bo, &lineRep)
	if err != nil {
		return nil, err
	}
	p.line = p.line[:0]
	// the count is stored as count - 1, but we're already reading the
	// first line, anyway.
	p.lineRep = int(lineRep)

	for len(p.line) < int(p.Header.CUPSBytesPerLine) {
		var n byte
		err := binary.Read(p.dec.r, p.dec.bo, &n)
		if err != nil {
			return nil, err
		}
		if n <= 127 {
			// n repeating colors
			n := int(n + 1)
			_, err := io.ReadFull(p.dec.r, p.color)
			if err != nil {
				return nil, err
			}

			for i := 0; i < n; i++ {
				p.line = append(p.line, p.color...)
			}
		} else {
			// n non-repeating colors
			n := 257 - int(n)
			for i := 0; i < n; i++ {
				_, err := io.ReadFull(p.dec.r, p.color)
				if err != nil {
					return nil, err
				}
				p.line = append(p.line, p.color...)
			}
		}
	}

	return p.line, nil
}

func (p *Page) ReadAll() ([]byte, error) {
	b := make([]byte, 0, p.Header.CUPSHeight*p.Header.CUPSBytesPerLine)
	for i := uint32(0); i < p.Header.CUPSHeight; i++ {
		line, err := p.ReadLine()
		if err != nil {
			return b, err
		}
		b = append(b, line...)
	}
	return b, nil
}

func cstring(b []byte) string {
	idx := bytes.IndexByte(b, 0)
	if idx < 0 {
		return ""
	}
	return string(b[:idx])
}

func (d *Decoder) readCString() string {
	if d.err != nil {
		return ""
	}
	b := make([]byte, 64)
	_, d.err = io.ReadFull(d.r, b)
	return cstring(b)
}

func (d *Decoder) readUint() uint32 {
	if d.err != nil {
		return 0
	}
	var v uint32
	d.err = binary.Read(d.r, d.bo, &v)
	return v
}

func (d *Decoder) readFloat() float32 {
	if d.err != nil {
		return 0
	}
	var v float32
	d.err = binary.Read(d.r, d.bo, &v)
	return v
}

func (d *Decoder) readBool() bool {
	return d.readUint() == 1
}

func (d *Decoder) decodeV1Header() (*PageHeader, error) {
	data := struct {
		AdvanceDistance  uint32
		AdvanceMedia     uint32
		Collate          uint32
		CutMedia         uint32
		Duplex           uint32
		HorizDPI         uint32
		VertDPI          uint32
		BoundingBox      BoundingBox
		InsertSheet      uint32
		Jog              uint32
		LeadingEdge      uint32
		MarginLeft       uint32
		MarginBottom     uint32
		ManualFeed       uint32
		MediaPosition    uint32
		MediaWeight      uint32
		MirrorPrint      uint32
		NegativePrint    uint32
		NumCopies        uint32
		Orientation      uint32
		OutputFaceUp     uint32
		Width            uint32
		Length           uint32
		Separations      uint32
		TraySwitch       uint32
		Tumble           uint32
		CUPSWidth        uint32
		CUPSHeight       uint32
		CUPSMediaType    uint32
		CUPSBitsPerColor uint32
		CUPSBitsPerPixel uint32
		CUPSBytesPerLine uint32
		CUPSColorOrder   uint32
		CUPSColorSpace   uint32
		CUPSCompression  uint32
		CUPSRowCount     uint32
		CUPSRowFeed      uint32
		CUPSRowStep      uint32
	}{}

	h := PageHeader{}
	h.MediaClass = d.readCString()
	h.MediaColor = d.readCString()
	h.MediaType = d.readCString()
	h.OutputType = d.readCString()

	// FIXME handle error
	err := binary.Read(d.r, d.bo, &data)
	if err != nil {
		return nil, err
	}
	h.AdvanceDistance = data.AdvanceDistance
	h.AdvanceMedia = int(data.AdvanceMedia)
	h.Collate = data.Collate == 1
	h.CutMedia = int(data.CutMedia)
	h.Duplex = data.Duplex == 1
	h.HorizDPI = data.HorizDPI
	h.VertDPI = data.VertDPI
	h.BoundingBox = data.BoundingBox
	h.InsertSheet = data.InsertSheet == 1
	h.Jog = int(data.Jog)
	h.LeadingEdge = int(data.LeadingEdge)
	h.MarginLeft = data.MarginLeft
	h.MarginBottom = data.MarginBottom
	h.ManualFeed = data.ManualFeed == 1
	h.MediaPosition = data.MediaPosition
	h.MediaWeight = data.MediaWeight
	h.MirrorPrint = data.MirrorPrint == 1
	h.NegativePrint = data.NegativePrint == 1
	h.NumCopies = data.NumCopies
	h.Orientation = int(data.Orientation)
	h.OutputFaceUp = data.OutputFaceUp == 1
	h.Width = data.Width
	h.Length = data.Length
	h.Separations = data.Separations == 1
	h.TraySwitch = data.TraySwitch == 1
	h.Tumble = data.Tumble == 1
	h.CUPSWidth = data.CUPSWidth
	h.CUPSHeight = data.CUPSHeight
	h.CUPSMediaType = data.CUPSMediaType
	h.CUPSBitsPerColor = data.CUPSBitsPerColor
	h.CUPSBitsPerPixel = data.CUPSBitsPerPixel
	h.CUPSBytesPerLine = data.CUPSBytesPerLine
	h.CUPSColorOrder = int(data.CUPSColorOrder)
	h.CUPSColorSpace = int(data.CUPSColorSpace)
	h.CUPSCompression = data.CUPSCompression
	h.CUPSRowCount = data.CUPSRowCount
	h.CUPSRowFeed = data.CUPSRowFeed
	h.CUPSRowStep = data.CUPSRowStep

	return &h, d.err
}

func (d *Decoder) decodeV2Header() (*PageHeader, error) {
	h, err := d.decodeV1Header()
	if err != nil {
		return nil, err
	}

	data := struct {
		CUPSNumColors               uint32
		CUPSBorderlessScalingFactor float32
		CUPSPageSize                [2]float32
		CUPSImagingBBox             CUPSBoundingBox
		CUPSInteger                 [16]uint32
		CUPSReal                    [16]float32
	}{}

	err = binary.Read(d.r, d.bo, &data)
	if err != nil {
		return nil, err
	}
	h.CUPSNumColors = data.CUPSNumColors
	h.CUPSBorderlessScalingFactor = data.CUPSBorderlessScalingFactor
	h.CUPSPageSize = data.CUPSPageSize
	h.CUPSImagingBBox = data.CUPSImagingBBox
	h.CUPSInteger = data.CUPSInteger
	h.CUPSReal = data.CUPSReal

	for i := range h.CUPSString {
		h.CUPSString[i] = d.readCString()
	}
	h.CUPSMarkerType = d.readCString()
	h.CUPSRenderingIntent = d.readCString()
	h.CUPSPageSizeName = d.readCString()

	return h, d.err
}

func bytesPerColor(h *PageHeader) (int, error) {
	switch h.CUPSColorOrder {
	case ChunkyPixels:
		return int(h.CUPSBitsPerPixel+7) / 8, nil
	case BandedPixels, PlanarPixels:
		return int(h.CUPSBitsPerColor+7) / 8, nil
	default:
		return 0, ErrUnknownColorOrder
	}
}
