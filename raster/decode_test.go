package raster

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func open(s string, t *testing.T) *os.File {
	f, err := os.Open(s)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestDecode(t *testing.T) {
	f, err := os.Open("testdata/raster")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.NextPage()
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(p.Header)
	b := make([]byte, p.TotalSize())
	err = p.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s", b)
}

func TestDecodeMultiplePages(t *testing.T) {
	f, err := os.Open("testdata/two_pages")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for {
		p, err := d.NextPage()
		if err == io.EOF {
			t.Logf("read %d pages", i)
			break
		}
		i++
		if err != nil {
			t.Fatal(err)
		}
		spew.Dump(p.Header)
		b := make([]byte, p.TotalSize())
		err = p.ReadAll(b)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%s", b)
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
		t.Errorf("got %v, want io.ErrUnexpectedEOF", err)
	}
}
