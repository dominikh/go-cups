package main

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"

	"honnef.co/go/cups/raster"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	d, err := raster.NewDecoder(f)
	if err != nil {
		log.Fatal(err)
	}

	p, err := d.NextPage()
	if err != nil {
		log.Fatal(err)
	}

	palette := color.Palette{color.Gray{Y: 255}, color.Gray{Y: 0}}
	img := image.NewPaletted(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: int(p.Header.CUPSWidth), Y: int(p.Header.CUPSHeight)},
	}, palette)

	b := make([]byte, p.LineSize())
	y := 0
	for {
		err := p.ReadLine(b)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		colors, err := p.ParseColors(b)
		if err != nil {
			log.Fatal(err)
		}
		for x, color := range colors {
			img.Set(x, y, color)
		}
		y++
	}

	err = png.Encode(os.Stdout, img)
	if err != nil {
		log.Fatal(err)
	}
}
