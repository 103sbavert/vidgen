package render

const (
	// lutBits sets the quantization of the per-pixel scalar (e.g. distance):
	// a frame's colors are tabulated at lutSize steps. 4096 steps keep the
	// error far below one 8-bit level even on the steepest ring transition.
	lutBits = 12
	lutSize = 1 << lutBits
	lutMask = lutSize - 1
)

// Table maps a quantized scalar index (see Field) to final yuv420p code
// values. Chroma is kept with two extra fractional bits so that averaging four
// samples (a 2x2 luma neighbourhood) rounds exactly once, at the end.
type Table struct {
	y [lutSize]uint8
	u [lutSize]uint16
	v [lutSize]uint16
}

// set stores the full-range RGB color (0..255) for index i as BT.709 YUV.
func (t *Table) set(i int, r, g, b float64) {
	y, u, v := yuv709(r, g, b)

	t.y[i] = uint8(y + 0.5)
	t.u[i] = uint16(u*4 + 0.5)
	t.v[i] = uint16(v*4 + 0.5)
}
