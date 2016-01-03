package raster

import (
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

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
	h, err := d.ReadPageHeader()
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(h)
	b, err := d.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s", b)
}
