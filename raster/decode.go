package raster

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/color"
	"io"
	"io/ioutil"
)

var (
	// ErrUnknownVersion is returned when encountering an unknown
	// magic byte sequence. It is indicative of input in a newer
	// format, or input that isn't a CUPS raster stream at all.
	ErrUnknownVersion = errors.New("unsupported file format or version")

	// ErrUnsupported is returned when encountering an unsupported
	// feature. This includes unsupported color spaces, color
	// orderings or bit depths.
	ErrUnsupported = errors.New("unsupported feature")

	// ErrBufferTooSmall is returned from ReadLine and ReadAll when
	// the buffer is smaller than Page.LineSize or Page.TotalSize
	// respectively.
	ErrBufferTooSmall = errors.New("buffer too small")

	// ErrInvalidFormat is returned when encountering values that
	// aren't possible in the supported versions of the format.
	ErrInvalidFormat = errors.New("error in the format")
)

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

type countingReader struct {
	r io.Reader
	n int
}

func (r *countingReader) Read(b []byte) (n int, err error) {
	n, err = r.r.Read(b)
	r.n += n
	return n, err
}

type Decoder struct {
	r       *countingReader
	bo      binary.ByteOrder
	err     error
	version int
	curPage *Page
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	d := &Decoder{r: &countingReader{r: r}}
	magic := make([]byte, 4)
	_, err := io.ReadFull(r, magic)
	if err != nil {
		return nil, err
	}
	var ok bool
	d.version, d.bo, ok = parseMagic(magic)
	if !ok {
		return nil, ErrUnknownVersion
	}
	return d, nil
}

type Page struct {
	Header    *Header
	dec       *Decoder
	line      []byte
	color     []byte
	lineRep   int
	linesRead int
}

// NextPage returns the next page in the raster stream. After a call
// to NextPage, all previously returned pages from this decoder cannot
// be used to decode image data anymore. Their header data, however,
// remains valid.
func (d *Decoder) NextPage() (*Page, error) {
	if d.curPage != nil {
		if err := d.curPage.discard(); err != nil {
			return nil, err
		}
	}
	var err error
	var h *Header

	n := d.r.n
	switch d.version {
	case 1:
		h, err = d.decodeV1Header()
	case 2, 3:
		h, err = d.decodeV2Header()
	default:
		// can't happen, NewDecoder rejects unknown versions
		panic("impossible")
	}
	if err == io.EOF && d.r.n != n {
		return nil, io.ErrUnexpectedEOF
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
	d.curPage = p
	return p, nil
}

func (p *Page) discard() error {
	n := p.UnreadLines() * int(p.LineSize())
	r := io.LimitReader(p.dec.r, int64(n))
	_, err := io.Copy(ioutil.Discard, r)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// ReadLine returns the next line of pixels in the image. It returns
// io.EOF if no more lines can be read. The buffer b must be at least
// p.Header.CUPSBytesPerLine bytes large.
func (p *Page) ReadLine(b []byte) error {
	if int64(len(b)) < int64(p.Header.CUPSBytesPerLine) {
		return ErrBufferTooSmall
	}
	if p.UnreadLines() == 0 {
		return io.EOF
	}
	p.linesRead++
	switch p.dec.version {
	case 1:
		return p.readRawLine(b)
	case 2:
		return p.readV2Line(b)
	case 3:
		return p.readRawLine(b)
	default:
		// can't happen, NewDecoder rejects unknown versions
		panic("impossible")
	}
}

// ReadLineColors reads a line and returns the color for each pixel.
// Unlike using ReadLine and ParseColors, this function will not
// return more values than there are pixels in a line. b is used as
// scratch space and must be at least p.Header.CUPSBytesPerLine bytes
// large.
func (p *Page) ReadLineColors(b []byte) ([]color.Color, error) {
	err := p.ReadLine(b)
	if err != nil {
		return nil, err
	}
	colors, err := p.ParseColors(b)
	if err != nil {
		return colors, err
	}
	if len(colors) > int(p.Header.CUPSWidth) {
		colors = colors[:p.Header.CUPSWidth]
	}
	return colors, nil
}

func (p *Page) readV2Line(b []byte) (err error) {
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if p.lineRep > 0 {
		p.lineRep--
		copy(b, p.line)
		return nil
	}

	var lineRep byte
	err = binary.Read(p.dec.r, p.dec.bo, &lineRep)
	if err != nil {
		return err
	}
	p.line = p.line[:0]
	// the count is stored as count - 1, but we're already reading the
	// first line, anyway.
	p.lineRep = int(lineRep)

	for len(p.line) < int(p.Header.CUPSBytesPerLine) {
		var n byte
		err := binary.Read(p.dec.r, p.dec.bo, &n)
		if err != nil {
			return err
		}
		if n <= 127 {
			// n repeating colors
			n := int(n + 1)
			_, err := io.ReadFull(p.dec.r, p.color)
			if err != nil {
				return err
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
					return err
				}
				p.line = append(p.line, p.color...)
			}
		}
	}
	copy(b, p.line)
	return nil
}

func (p *Page) readRawLine(b []byte) error {
	b = b[:p.Header.CUPSBytesPerLine]
	_, err := io.ReadFull(p.dec.r, b)
	return err
}

// UnreadLines returns the number of unread lines in the page.
func (p *Page) UnreadLines() int {
	return int(p.Header.CUPSHeight) - p.linesRead
}

// ReadAll reads the entire page into b. If ReadLine has been called
// previously, ReadAll will read the remainder of the page. It returns
// io.EOF if the entire page has been read already.
func (p *Page) ReadAll(b []byte) error {
	if len(b) < int(p.LineSize())*p.UnreadLines() {
		return ErrBufferTooSmall
	}
	n := p.UnreadLines()
	if n == 0 {
		return io.EOF
	}
	for i := uint32(0); i < uint32(n); i++ {
		start := i * p.Header.CUPSBytesPerLine
		end := start + p.Header.CUPSBytesPerLine
		err := p.ReadLine(b[start:end:end])
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadAllColors reads the page and returns the color for each pixel.
// Unlike using ReadAll and ParseColors, this function will not
// return more values than there are pixels in a page. b is used as
// scratch space and must be at least p.Header.CUPSBytesPerLine bytes
// large.
func (p *Page) ReadAllColors(b []byte) ([]color.Color, error) {
	if int64(len(b)) < int64(p.Header.CUPSBytesPerLine) {
		return nil, ErrBufferTooSmall
	}
	n := p.UnreadLines()
	if n == 0 {
		return nil, io.EOF
	}
	var out []color.Color
	for i := 0; i < n; i++ {
		colors, err := p.ReadLineColors(b)
		if err == io.EOF {
			return out, io.ErrUnexpectedEOF
		}
		if err != nil {
			return out, err
		}
		out = append(out, colors...)
	}
	return out, nil
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

func (d *Decoder) decodeV1Header() (*Header, error) {
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

	h := Header{}
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

func (d *Decoder) decodeV2Header() (*Header, error) {
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

func bytesPerColor(h *Header) (int, error) {
	switch h.CUPSColorOrder {
	case ChunkyPixels:
		return int(h.CUPSBitsPerPixel+7) / 8, nil
	case BandedPixels, PlanarPixels:
		return int(h.CUPSBitsPerColor+7) / 8, nil
	default:
		// The versions that we support only know these 3 color orders
		return 0, ErrInvalidFormat
	}
}
