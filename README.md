# Fast Bilateral filter for Golang

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/mdouchement/bilateral)
[![Go Report Card](https://goreportcard.com/badge/github.com/mdouchement/bilateral)](https://goreportcard.com/report/github.com/mdouchement/bilateral)
[![License](https://img.shields.io/github/license/mdouchement/bilateral.svg)](http://opensource.org/licenses/MIT)

A FastBilateral filter is a non-linear, edge-preserving and noise-reducing
smoothing filter for images. The intensity value at each pixel in an image is
replaced by a weighted average of intensity values from nearby pixels.

Algorithm and implementation is based on http://people.csail.mit.edu/sparis/bf/ <br>
Please cite above paper for research purpose.

| Before | After
|:--:|:--:|
| ![before](https://github.com/mdouchement/bilateral/blob/master/data/greekdome-gray.jpeg) | ![after](https://github.com/mdouchement/bilateral/blob/master/data/greekdome-gray-filtered.jpeg) |
| ![before](https://github.com/mdouchement/bilateral/blob/master/data/greekdome.jpeg) | ![after](https://github.com/mdouchement/bilateral/blob/master/data/greekdome-filtered.jpeg) |

> Sigma Space: 16 <br>
> Sigma Range/Color: 0.1

## Requirements

- Golang 1.7.x

## Installation

```bash
$ go get -u github.com/mdouchement/bilateral
```

## Usage

```go
fi, _ := os.Open("input_path")
defer fi.Close()

m, _, _ := image.Decode(fi)

start := time.Now()
fbl := bilateral.NewFastBilateral(m, 16, 0.1)
fbl.Execute()
m2 := fbl.ResultImage() // Or use `At(x, y)` func or just use `fbl` as an image.Image for chained treatments.

fo, _ := os.Create("output_path")
defer fo.Close()

jpeg.Encode(fo, m2, &jpeg.Options{Quality: 100})
```

[Full example](https://github.com/mdouchement/bilateral/blob/master/data/main.go)

## Licence

MIT. See the [LICENSE](https://github.com/mdouchement/bilateral/blob/master/LICENSE) for more details.

## Contributing

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Launch linter (gometalinter --config=gometalinter.json ./...)
4. Commit your changes (git commit -am 'Add some feature')
5. Push to the branch (git push origin my-new-feature)
6. Create new Pull Request
