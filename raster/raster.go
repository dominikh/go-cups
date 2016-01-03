package raster

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

type PageHeader struct {
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
