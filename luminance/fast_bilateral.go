package luminance

import (
	"image"
	"image/color"
	"math"
	"sync"

	colorful "github.com/lucasb-eyer/go-colorful"
	"gonum.org/v1/gonum/mat"
)

const (
	maxrange  = 65535
	dimension = 3
	// padding space
	paddingS = 2
	// padding range (luminance)
	paddingR = 2
)

// A FastBilateral filter is a non-linear, edge-preserving and noise-reducing
// smoothing filter for images. The intensity value at each pixel in an image is
// replaced by a weighted average of intensity values from nearby pixels.
type FastBilateral struct {
	Image      image.Image
	SigmaRange float64
	SigmaSpace float64
	minmaxOnce sync.Once
	min        float64
	max        float64
	// size:
	// 0 -> smallWidth
	// 1 -> smallHeight
	// 2 -> smallLuminance
	size []int
	grid *mat.Dense
	auto bool
}

// Auto instanciates a new FastBilateral with automatic sigma values.
func Auto(m image.Image) *FastBilateral {
	f := New(m, 16, 0.1)
	f.auto = true
	return f
}

// New instanciates a new FastBilateral.
func New(m image.Image, sigmaSpace, sigmaRange float64) *FastBilateral {
	return &FastBilateral{
		Image:      m,
		SigmaRange: sigmaRange,
		SigmaSpace: sigmaSpace,
		min:        math.Inf(1),
		max:        math.Inf(-1),
		size:       make([]int, dimension),
	}
}

// Execute runs the bilateral filter.
func (f *FastBilateral) Execute() {
	f.minmaxOnce.Do(f.minmax)
	f.downsampling()
	f.convolution()
	f.normalize()
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
	r, g, b, a := f.Image.At(x, y).RGBA()
	X, Y, Z := colorful.LinearRgbToXyz(f.color(r), f.color(g), f.color(b))

	// Grid coords
	gw := float64(x)/f.SigmaSpace + paddingS // Grid width
	gh := float64(y)/f.SigmaSpace + paddingS // Grid height
	gc := (Y-f.min)/f.SigmaRange + paddingR  // Grid luminance
	Y2 := f.trilinearInterpolation(gw, gh, gc)

	delta := Y - Y2
	R, G, B := colorful.XyzToLinearRgb(X-delta, Y2, Z-delta)
	return color.RGBA{
		R: uint8(f.clamp(0, 255, int(R*255))),
		G: uint8(f.clamp(0, 255, int(G*255))),
		B: uint8(f.clamp(0, 255, int(B*255))),
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
	d := f.Image.Bounds()
	for y := 0; y < d.Dy(); y++ {
		for x := 0; x < d.Dx(); x++ {
			r, g, b, _ := f.Image.At(x, y).RGBA()
			_, Y, _ := colorful.LinearRgbToXyz(f.color(r), f.color(g), f.color(b))
			f.min = math.Min(f.min, Y)
			f.max = math.Max(f.max, Y)
		}
	}

	if f.auto {
		f.SigmaRange = (f.max - f.min) * 0.1
	}

	f.size[0] = int(float64(d.Dx()-1)/f.SigmaSpace) + 1 + 2*paddingS
	f.size[1] = int(float64(d.Dy()-1)/f.SigmaSpace) + 1 + 2*paddingS
	f.size[2] = int((f.max-f.min)/f.SigmaRange) + 1 + 2*paddingR

	// fmt.Println("ssp:", f.SigmaSpace, " - sra:", f.SigmaRange)
	// fmt.Println("min:", f.min, "- max:", f.max)
	// fmt.Println("size:", f.mul(f.size...), f.size)
}

func (f *FastBilateral) downsampling() {
	d := f.Image.Bounds()
	offset := make([]int, dimension)

	size := f.mul(f.size...)
	dim := dimension - 1 // # 1 luminance and 1 threshold (edge weight)
	f.grid = mat.NewDense(size, dim, make([]float64, dim*size))

	for x := 0; x < d.Dx(); x++ {
		offset[0] = int(float64(x)/f.SigmaSpace+0.5) + paddingS

		for y := 0; y < d.Dy(); y++ {
			offset[1] = int(float64(y)/f.SigmaSpace+0.5) + paddingS

			r, g, b, _ := f.Image.At(x, y).RGBA()
			_, Y, _ := colorful.LinearRgbToXyz(f.color(r), f.color(g), f.color(b))

			offset[2] = int((Y-f.min)/f.SigmaRange+0.5) + paddingR

			i := f.offset(offset...)
			v := f.grid.RawRowView(i)
			v[0] += Y // luminance
			v[1]++    // threshold
			f.grid.SetRow(i, v)
		}
	}
}

func (f *FastBilateral) convolution() {
	size := f.mul(f.size...)
	dim := dimension - 1 // # luminance and 1 threshold (edge weight)
	buffer := mat.NewDense(size, dim, make([]float64, dim*size))

	for dim := 0; dim < dimension; dim++ { // x, y, and luminance
		off := make([]int, dimension)
		off[dim] = 1 // Wanted dimension offset

		for n := 0; n < 2; n++ { // itterations (pass?)
			f.grid, buffer = buffer, f.grid

			for x := 1; x < f.size[0]-1; x++ {
				for y := 1; y < f.size[1]-1; y++ {

					for z := 1; z < f.size[2]-1; z++ {
						vg := f.grid.RowView(f.offset(x, y, z)).(*mat.VecDense)
						prev := buffer.RowView(f.offset(x-off[0], y-off[1], z-off[2])).(*mat.VecDense)
						curr := buffer.RowView(f.offset(x, y, z)).(*mat.VecDense)
						next := buffer.RowView(f.offset(x+off[0], y+off[1], z+off[2])).(*mat.VecDense)

						// (prev + 2.0 * curr + next) / 4.0
						vg.AddVec(prev, next)
						vg.AddScaledVec(vg, 2, curr)
						vg.ScaleVec(0.25, vg)
					}
				}
			}
		}
	}
	return
}

func (f *FastBilateral) normalize() {
	r, _ := f.grid.Dims()
	for i := 0; i < r; i++ {
		if threshold := f.grid.At(i, 1); threshold != 0 {
			f.grid.Set(i, 0, f.grid.At(i, 0)/threshold)
		}
	}
}

func (f *FastBilateral) trilinearInterpolation(gx, gy, gz float64) float64 {
	width := f.size[0]
	height := f.size[1]
	depth := f.size[2]

	// Index
	x := f.clamp(0, width-1, int(gx))
	xx := f.clamp(0, width-1, x+1)
	y := f.clamp(0, height-1, int(gy))
	yy := f.clamp(0, height-1, y+1)
	z := f.clamp(0, depth-1, int(gz))
	zz := f.clamp(0, depth-1, z+1)

	// Alpha
	xa := gx - float64(x)
	ya := gy - float64(y)
	za := gz - float64(z)

	// Interpolation
	return (1.0-ya)*(1.0-xa)*(1.0-za)*f.grid.At(f.offset(x, y, z), 0) +
		(1.0-ya)*xa*(1.0-za)*f.grid.At(f.offset(xx, y, z), 0) +
		ya*(1.0-xa)*(1.0-za)*f.grid.At(f.offset(x, yy, z), 0) +
		ya*xa*(1.0-za)*f.grid.At(f.offset(xx, yy, z), 0) +
		(1.0-ya)*(1.0-xa)*za*f.grid.At(f.offset(x, y, zz), 0) +
		(1.0-ya)*xa*za*f.grid.At(f.offset(xx, y, zz), 0) +
		ya*(1.0-xa)*za*f.grid.At(f.offset(x, yy, zz), 0) +
		ya*xa*za*f.grid.At(f.offset(xx, yy, zz), 0)
}

func (f *FastBilateral) clamp(min, max, v int) int {
	if v < min {
		v = 0
	}
	if v > max {
		v = max
	}
	return v
}

func (f *FastBilateral) mul(size ...int) (n int) {
	n = 1
	for _, v := range size {
		n *= v
	}
	return
}

// slice[x + WIDTH*y + WIDTH*HEIGHT*z)]
func (f *FastBilateral) offset(size ...int) (n int) {
	n = size[0] // x
	for i, v := range size[1:] {
		n += v * f.mul(f.size[0:i+1]...) // y, z
	}
	return
}

func (f *FastBilateral) color(v uint32) float64 {
	return float64(v) / maxrange
}
