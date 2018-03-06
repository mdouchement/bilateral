package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"time"

	"github.com/mdouchement/bilateral"
	"github.com/mdouchement/bilateral/luminance"
)

var (
	entries = []map[string]string{
		{"type": "rgb", "name": "greekdome-gray", "in": "./greekdome-gray.jpeg", "out": "./greekdome-gray-filtered.jpeg"},
		{"type": "rgb", "name": "greekdome-rgb", "in": "./greekdome.jpeg", "out": "./greekdome-filtered.jpeg"},
		{"type": "lum", "name": "greekdome-gray-lum", "in": "./greekdome-gray.jpeg", "out": "./greekdome-gray-filtered-lum.jpeg"},
		{"type": "lum", "name": "greekdome-rgb-lum", "in": "./greekdome.jpeg", "out": "./greekdome-filtered-lum.jpeg"},
	}
)

func main() {
	for _, entry := range entries {
		fi, err := os.Open(entry["in"])
		check(err)
		defer fi.Close()

		m, _, err := image.Decode(fi)
		check(err)

		fmt.Println(entry["name"], " bounds:", m.Bounds().Dx(), m.Bounds().Dy())

		var m2 image.Image
		start := time.Now()
		if entry["type"] == "lum" {
			fbl := luminance.NewFastBilateralAuto(m)
			fbl.Execute()
			m2 = fbl.ResultImage()
		} else {
			fbl := bilateral.NewFastBilateralAuto(m)
			fbl.Execute()
			m2 = fbl.ResultImage()
		}
		fmt.Printf("%s takes %v\n", entry["name"], time.Now().Sub(start))

		fo, err := os.Create(entry["out"])
		check(err)
		defer fo.Close()

		err = jpeg.Encode(fo, m2, &jpeg.Options{Quality: 100})
		check(err)

		fmt.Println("----")
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
