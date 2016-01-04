package raster

import (
	"io"
	"os"
	"testing"
)

func open(s string, t *testing.T) *os.File {
	f, err := os.Open(s)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestDecodeMultiplePages(t *testing.T) {
	f := open("testdata/two_pages", t)
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
		b := make([]byte, p.TotalSize())
		err = p.ReadAll(b)
		if err != nil {
			t.Errorf("got error %q reading page, want nil", err)
		}
	}
}

func TestDecodeTruncatedLine(t *testing.T) {
	f := open("testdata/raster_truncated", t)
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
	f := open("testdata/raster_missing_line", t)
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.NextPage()
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, p.TotalSize())
	err = p.ReadAll(b)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecodeTruncatedHeader(t *testing.T) {
	f := open("testdata/truncated_header", t)
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
