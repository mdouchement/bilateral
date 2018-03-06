package bilateral

import (
	"image"
	"image/color"
	"math"
	"math/big"
	"sync"

	"gonum.org/v1/gonum/mat"
)

const (
	// padding space
	paddingS = 2
	// padding range (color)
	paddingR = 2
	// colors' index
	c1 = 0
	c2 = 1
	c3 = 2
)

// A FastBilateral filter is a non-linear, edge-preserving and noise-reducing
// smoothing filter for images. The intensity value at each pixel in an image is
// replaced by a weighted average of intensity values from nearby pixels.
type FastBilateral struct {
	Image      image.Image
	SigmaRange float64
	SigmaSpace float64
	dimension  int
	minmaxOnce sync.Once
	min        []float64
	max        []float64
	// Grid size:
	// 0 -> smallWidth
	// 1 -> smallHeight
	// 2 -> smallColor1Depth (gray & color)
	// 3 -> smallColor2Depth (color)
	// 4 -> smallColor3Depth (color)
	size []int
	grid *grid
	auto bool
}

// NewFastBilateralAuto instanciates a new FastBilateral with automatic sigma values.
func NewFastBilateralAuto(m image.Image) *FastBilateral {
	f := NewFastBilateral(m, 16, 0.1)
	f.auto = true
	return f
}

// NewFastBilateral instanciates a new FastBilateral.
func NewFastBilateral(img image.Image, sigmaSpace, sigmaRange float64) *FastBilateral {
	dimension := 5 // default: x, y, colors (z1, z2, z3)
	fbl := &FastBilateral{
		Image:      img,
		SigmaRange: sigmaRange,
		SigmaSpace: sigmaSpace,
		dimension:  dimension,
		min:        make([]float64, dimension-2),
		max:        make([]float64, dimension-2),
		size:       make([]int, dimension),
	}
	for i := range fbl.min {
		fbl.min[i] = math.Inf(1)
		fbl.max[i] = math.Inf(-1)
	}

	return fbl
}

// Execute runs the bilateral filter.
func (f *FastBilateral) Execute() {
	f.minmaxOnce.Do(f.minmax)
	f.downsampling()
	f.convolution()
}

// ColorModel returns the Image's color model.
func (f *FastBilateral) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds implements image.Image interface.
func (f *FastBilateral) Bounds() image.Rectangle {
	return f.Image.Bounds()
}

// At computes the interpolation and returns the filtered color at the given coordinates.
func (f *FastBilateral) At(x, y int) color.Color {
	pixel := f.Image.At(x, y)
	r, g, b, a := pixel.RGBA()
	rgb := []float64{fcolor(r), fcolor(g), fcolor(b)}

	offset := make([]float64, f.dimension)
	// Grid coords
	offset[0] = float64(x)/f.SigmaSpace + paddingS // Grid width
	offset[1] = float64(y)/f.SigmaSpace + paddingS // Grid height
	for z := 0; z < f.dimension-2; z++ {
		offset[2+z] = (rgb[z]-f.min[z])/f.SigmaRange + paddingR // Grid color
	}

	c := f.nLinearInterpolation(offset...)
	c.colors.ScaleVec(1/c.threshold, c.colors) // Normalize

	len := c.colors.Len()
	channel := func(z int) uint8 {
		if z < len {
			return uint8(clamp(0, 255, int(c.colors.AtVec(z)*255)))
		}
		return uint8(clamp(0, 255, int(c.colors.AtVec(len-1)*255)))
	}
	return color.RGBA{
		R: channel(c1),
		G: channel(c2),
		B: channel(c3),
		A: uint8(a),
	}
}

// ResultImage computes the interpolation and returns the filtered image.
func (f *FastBilateral) ResultImage() image.Image {
	d := f.Image.Bounds()
	dst := image.NewRGBA(d)
	for x := 0; x < d.Dx(); x++ {
		for y := 0; y < d.Dy(); y++ {
			dst.Set(x, y, f.At(x, y))
		}
	}
	return dst
}

func (f *FastBilateral) minmax() {
	gray := true
	d := f.Image.Bounds()
	for y := 0; y < d.Dy(); y++ {
		for x := 0; x < d.Dx(); x++ {
			pixel := f.Image.At(x, y)
			r, g, b, _ := pixel.RGBA()
			if gray && (r != g || g != b) {
				gray = false
			}
			for ci, c := range []uint32{r, g, b} {
				c64 := fcolor(c)
				f.min[ci] = math.Min(f.min[ci], c64)
				f.max[ci] = math.Max(f.max[ci], c64)
			}
		}
	}

	if gray {
		// Go to gray scale to spped up the algo
		f.dimension = 3 // x, y, z
		f.size = f.size[0:f.dimension]
		f.min = f.min[0:f.dimension]
		f.max = f.max[0:f.dimension]
	}

	if f.auto {
		min := math.Inf(1)
		max := math.Inf(-1)
		for n := 0; n < f.dimension-2; n++ {
			min = math.Min(min, f.min[n])
			max = math.Max(max, f.max[n])
		}
		f.SigmaRange = (max - min) * 0.1
	}

	f.size[0] = int(float64(d.Dx()-1)/f.SigmaSpace) + 1 + 2*paddingS
	f.size[1] = int(float64(d.Dy()-1)/f.SigmaSpace) + 1 + 2*paddingS
	for c := 0; c < f.dimension-2; c++ {
		f.size[2+c] = int((f.max[c]-f.min[c])/f.SigmaRange) + 1 + 2*paddingR
	}

	// fmt.Println("ssp:", f.SigmaSpace, " - sra:", f.SigmaRange)
	// fmt.Println("min:", f.min, "- max:", f.max)
	// fmt.Println("size:", mul(f.size...), f.size)
}

func (f *FastBilateral) downsampling() {
	d := f.Image.Bounds()
	offset := make([]int, f.dimension)

	dim := f.dimension - 2
	f.grid = newGrid(f.size, dim)

	for x := 0; x < d.Dx(); x++ {
		offset[0] = int(1*float64(x)/f.SigmaSpace+0.5) + paddingS

		for y := 0; y < d.Dy(); y++ {
			offset[1] = int(1*float64(y)/f.SigmaSpace+0.5) + paddingS

			pixel := f.Image.At(x, y)
			r, g, b, _ := pixel.RGBA()
			rgb := []float64{fcolor(r), fcolor(g), fcolor(b)}

			for z := 0; z < f.dimension-2; z++ {
				offset[2+z] = int((rgb[z]-f.min[z])/f.SigmaRange+0.5) + paddingR
			}

			v := f.grid.At(offset...)
			v.colors.AddVec(v.colors, mat.NewVecDense(dim, rgb[0:f.dimension-2]))
			v.threshold++
		}
	}
}

func (f *FastBilateral) convolution() {
	dim := f.dimension - 2
	buffer := newGrid(f.size, dim)

	var vg *cell
	var prev *cell
	var curr *cell
	var next *cell

	for dim := 0; dim < f.dimension; dim++ { // x, y, and colors depths
		off := make([]int, f.dimension)
		off[dim] = 1 // Wanted dimension offset

		for n := 0; n < 2; n++ { // itterations (pass?)
			f.grid, buffer = buffer, f.grid

			for x := 1; x < f.size[0]-1; x++ {
				for y := 1; y < f.size[1]-1; y++ {

					for z1 := 1; z1 < f.size[2+c1]-1; z1++ {
						if f.dimension == 5 {
							for z2 := 1; z2 < f.size[2+c2]-1; z2++ {
								for z3 := 1; z3 < f.size[2+c3]-1; z3++ {
									vg = f.grid.At(x, y, z1, z2, z3)
									prev = buffer.At(x-off[0], y-off[1], z1-off[2], z2-off[3], z3-off[4])
									curr = buffer.At(x, y, z1, z2, z3)
									next = buffer.At(x+off[0], y+off[1], z1+off[2], z2+off[3], z3+off[4])

									// (prev + 2.0 * curr + next) / 4.0
									vg.Add(prev, next)
									vg.AddScaled(vg, 2, curr)
									vg.Scale(0.25, vg)
								}
							}
						} else {
							vg = f.grid.At(x, y, z1)
							prev = buffer.At(x-off[0], y-off[1], z1-off[2])
							curr = buffer.At(x, y, z1)
							next = buffer.At(x+off[0], y+off[1], z1+off[2])

							// (prev + 2.0 * curr + next) / 4.0
							vg.Add(prev, next)
							vg.AddScaled(vg, 2, curr)
							vg.Scale(0.25, vg)
						}
					}
				}
			}
		}
	}
	return
}

// Perform linear interpolation.
// For 3 dimensions, it will perform this static algo:
//
// func (f *FastBilateral) trilinearInterpolation(gx, gy, gz float64) float64 {
// 	width := f.size[0]
// 	height := f.size[1]
// 	depth := f.size[2+c1]
//
// 	// Index
// 	x := clamp(0, width-1, int(gx))
// 	xx := clamp(0, width-1, x+1)
// 	y := clamp(0, height-1, int(gy))
// 	yy := clamp(0, height-1, y+1)
// 	z := clamp(0, depth-1, int(gz))
// 	zz := clamp(0, depth-1, z+1)
//
// 	// Alpha
// 	xa := gx - float64(x)
// 	ya := gy - float64(y)
// 	za := gz - float64(z)
//
// 	// Interpolation
// 	return (1.0-ya)*(1.0-xa)*(1.0-za)*f.grid.At(x, y, z).colors.At(c1, 0) +
// 		(1.0-ya)*xa*(1.0-za)*f.grid.At(xx, y, z).colors.At(c1, 0) +
// 		ya*(1.0-xa)*(1.0-za)*f.grid.At(x, yy, z).colors.At(c1, 0) +
// 		ya*xa*(1.0-za)*f.grid.At(xx, yy, z).colors.At(c1, 0) +
// 		(1.0-ya)*(1.0-xa)*za*f.grid.At(x, y, zz).colors.At(c1, 0) +
// 		(1.0-ya)*xa*za*f.grid.At(xx, y, zz).colors.At(c1, 0) +
// 		ya*(1.0-xa)*za*f.grid.At(x, yy, zz).colors.At(c1, 0) +
// 		ya*xa*za*f.grid.At(xx, yy, zz).colors.At(c1, 0)
// }
func (f *FastBilateral) nLinearInterpolation(offset ...float64) *cell {
	permutations := 1 << uint(f.dimension)
	index := make([]int, f.dimension)
	indexx := make([]int, f.dimension)
	alpha := make([]float64, f.dimension)

	for n, s := range f.size {
		off := offset[n]
		size := s - 1
		index[n] = clamp(0, size, int(off))
		indexx[n] = clamp(0, size, index[n]+1)
		alpha[n] = off - float64(index[n])
	}

	// Interpolation
	c := &cell{colors: mat.NewVecDense(f.dimension-2, nil)}
	bitset := big.NewInt(int64(0)) // Use to perform all the interpolation's permutations
	off := make([]int, f.dimension)
	var scale float64
	for i := 0; i < permutations; i++ {
		bitset.SetUint64(uint64(i))
		scale = 1.0
		for n := 0; n < f.dimension; n++ {
			if bitset.Bit(n) == 1 {
				off[n] = index[n]
				scale *= 1.0 - alpha[n]
			} else {
				off[n] = indexx[n]
				scale *= alpha[n]
			}
		}
		c.AddScaled(c, scale, f.grid.At(off...))
	}

	return c
}
