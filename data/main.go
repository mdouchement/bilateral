package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"time"

	_ "github.com/mdouchement/thdr/ppm"

	"github.com/mdouchement/bilateral"
)

var (
	entries = []map[string]string{
		{"name": "greekdome-gray", "in": "./greekdome-gray.jpeg", "out": "./greekdome-gray-filtered.jpeg"},
		{"name": "greekdome-rgb", "in": "./greekdome.jpeg", "out": "./greekdome-filtered.jpeg"},
	}
)

func main() {
	for _, entry := range entries {
		fi, err := os.Open(entry["in"])
		check(err)
		defer fi.Close()

		m, _, err := image.Decode(fi)
		check(err)

		fmt.Println("Bounds:", m.Bounds().Dx(), m.Bounds().Dy())

		start := time.Now()
		fbl := bilateral.NewFastBilateral(m, 16, 0.1)
		fbl.Execute()
		m2 := fbl.ResultImage()
		fmt.Printf("%s takes %v\n", entry["name"], time.Now().Sub(start))

		fo, err := os.Create(entry["out"])
		check(err)
		defer fo.Close()

		err = jpeg.Encode(fo, m2, &jpeg.Options{Quality: 100})
		check(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
