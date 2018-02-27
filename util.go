package bilateral

const maxrange = 65535

func clamp(min, max, v int) int {
	if v < min {
		v = 0
	}
	if v > max {
		v = max
	}
	return v
}

func mul(size ...int) (n int) {
	n = 1
	for _, v := range size {
		if v != 0 {
			n *= v
		}
	}
	return
}

func fcolor(v uint32) float64 {
	return float64(v) / maxrange
}
