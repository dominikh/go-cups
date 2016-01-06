package raster

import (
	"compress/gzip"
	"io"
	"os"
	"testing"
)

type file struct {
	f *os.File
	r *gzip.Reader
}

func (f file) Read(b []byte) (int, error) {
	return f.r.Read(b)
}

func (f file) Close() error {
	return f.f.Close()
}

func open(s string, t *testing.T) io.ReadCloser {
	f, err := os.Open("testdata/" + s + ".gz")
	if err != nil {
		t.Fatal(err)
	}

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	return file{f, gz}
}

func TestDecode(t *testing.T) {
	const width = 633
	const height = 633
	const size = width * height
	var tests = []struct {
		file      string
		openErr   error
		parseErr  error
		numColors int
	}{
		// The width of the image is 633 pixels. However, we store 8
		// pixels in one byte, so the byte for the last pixel in each
		// line contains 7 extra bits.
		{"gradient_chunked_k_1_1", nil, nil, size + 7*height},
		{"gradient_chunked_k_8_8", nil, nil, size},
		{"gradient_chunked_cmyk_8_32", nil, nil, size},
		{"gradient_chunked_cmyk_1_4", nil, ErrUnsupported, size},
		{"garbage", ErrUnknownVersion, nil, 1e4},
	}

	for _, tt := range tests {
		f := open(tt.file, t)
		defer f.Close()
		d, err := NewDecoder(f)
		if err != tt.openErr {
			t.Errorf("NewDecoder(%s): got err %v, want %v", tt.file, err, tt.openErr)
		}
		if err != nil {
			continue
		}
		p, err := d.NextPage()
		if err != nil {
			t.Errorf("%s: d.NextPage(): got err %v, want nil", tt.file, err)
			continue
		}
		b := make([]byte, p.Size())
		err = p.ReadAll(b)
		if err != nil {
			t.Errorf("%s: p.ReadAll(): got err %v, want nil", tt.file, err)
			continue
		}
		colors, err := p.ParseColors(b)
		if err != tt.parseErr {
			t.Errorf("%s: p.ParseColors(): got err %v, want %v", tt.file, err, tt.parseErr)
		}
		if err != nil {
			continue
		}
		if len(colors) != tt.numColors {
			t.Errorf("%s: parsed %d colors, want %d", tt.file, len(colors), tt.numColors)
		}
	}
}

func TestDecodeMultiplePages(t *testing.T) {
	f := open("two_pages", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for {
		p, err := d.NextPage()
		if err == io.EOF {
			break
		}
		i++
		if err != nil {
			t.Errorf("got error %q advancing page, want nil", err)
		}
		b := make([]byte, p.Size())
		err = p.ReadAll(b)
		if err != nil {
			t.Errorf("got error %q reading page, want nil", err)
		}
	}
	if i != 2 {
		t.Errorf("read %d pages, want 2", i)
	}
}

func TestDecodeSkipPage(t *testing.T) {
	f := open("two_pages", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.NextPage()
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, p.LineSize())
	if err = p.ReadLine(b); err != nil {
		t.Fatal(err)
	}
	p, err = d.NextPage()
	if err != nil {
		t.Errorf("NextPage after a partially read page failed with %v", err)
	}
}

func TestDecodeSkipTruncatedPage(t *testing.T) {
	f := open("raster_truncated", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.NextPage()
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, p.LineSize())
	if err = p.ReadLine(b); err != nil {
		t.Fatal(err)
	}
	err = p.discard()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("skipping over partially read truncated page returned %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecodeTruncatedLine(t *testing.T) {
	f := open("raster_truncated", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.NextPage()
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, p.LineSize())
	lines := p.UnreadLines()
	var i int
	for i = 1; i <= lines; i++ {
		err = p.ReadLine(b)
		if err != nil {
			break
		}
	}
	const brokenLine = 236
	if err != nil && i != brokenLine {
		t.Errorf("got read error %q after %d iterations, expected %d iterations", err, i, brokenLine)
	}
	if err != io.ErrUnexpectedEOF {
		t.Errorf("got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecodeMissingLine(t *testing.T) {
	t.Skip("TODO(dh): provide fixture")
	f := open("raster_missing_line", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.NextPage()
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, p.Size())
	err = p.ReadAll(b)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecodeTruncatedHeader(t *testing.T) {
	f := open("truncated_header", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.NextPage()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("got %q, want io.ErrUnexpectedEOF", err)
	}
}
