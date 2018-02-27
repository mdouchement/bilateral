package bilateral

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

const (
	xi = 0
	yi = 1
	zi = 2
)

type (
	// Convenient matrix for FastBilateral filter.
	grid struct {
		size  []int       // X, Y & Zs' dimensions
		cells [][][]*cell // X, Y & Zs coords
	}
	// An entry of the grid
	cell struct {
		colors    *mat.VecDense
		threshold float64 // aka image edges
	}
)

func newGrid(size []int, n int) *grid {
	if len(size) < 3 {
		panic("Grid size must be greater or equals to 3")
	}

	cells := make([][][]*cell, size[xi])
	for x := range cells {
		cells[x] = make([][]*cell, size[yi])
		for y := range cells[x] {
			cells[x][y] = make([]*cell, mul(size[zi:]...))
			for z := range cells[x][y] {
				cells[x][y][z] = &cell{colors: mat.NewVecDense(n, nil)}
			}
		}
	}

	return &grid{
		size:  size,
		cells: cells,
	}
}

func (g *grid) At(offsets ...int) *cell {
	// 1D array
	// offset := offsets[0] // x
	// for i, v := range offsets[1:] {
	// 	offset += v * mul(g.size[0:i+1]...) // y, z, ...
	// }
	//
	// return g.cells[offset] // i.e slice[x + WIDTH*y + WIDTH*HEIGHT*z)]

	offset := offsets[zi] // z1
	for i, v := range offsets[zi+1:] {
		offset += v * mul(g.size[zi:zi+i+1]...) // z2, zi...
	}
	return g.cells[offsets[xi]][offsets[yi]][offset]
}

func (c *cell) Add(a, b *cell) {
	c.colors.AddVec(a.colors, b.colors)
	c.threshold = a.threshold + b.threshold
}

func (c *cell) Scale(alpha float64, a *cell) {
	c.colors.ScaleVec(alpha, a.colors)
	c.threshold = alpha * a.threshold
}

func (c *cell) AddScaled(a *cell, alpha float64, b *cell) {
	c.colors.AddScaledVec(a.colors, alpha, b.colors)
	c.threshold = a.threshold + alpha*b.threshold
}

func (c *cell) Copy() *cell {
	return &cell{
		colors:    mat.VecDenseCopyOf(c.colors),
		threshold: c.threshold,
	}
}

func (c *cell) String() string {
	return fmt.Sprintf("[c: %v t: %f]", c.colors.RawVector().Data, c.threshold)
}
