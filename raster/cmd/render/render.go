package main

import (
	"image/png"
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

	// palette := color.Palette{color.Gray{Y: 255}, color.Gray{Y: 0}}
	// img2 := image.NewPaletted(image.Rectangle{
	// 	Min: image.Point{X: 0, Y: 0},
	// 	Max: image.Point{X: int(p.Header.CUPSWidth), Y: int(p.Header.CUPSHeight)},
	// }, palette)

	img := p.Image()
	// img.(*raster.Image).Render2(img2)

	// err = p.Render(img2)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	err = png.Encode(os.Stdout, img)
	if err != nil {
		log.Fatal(err)
	}
}
