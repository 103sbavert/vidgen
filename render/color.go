package render

// BT.709 luma coefficients.
const (
	kr = 0.2126
	kb = 0.0722
	kg = 1 - kr - kb
)

// yuv709 converts full-range RGB (0..255) into limited-range ("tv") BT.709
// Y'CbCr in 8-bit code values (Y 16..235, Cb/Cr 16..240), unrounded.
func yuv709(r, g, b float64) (y, u, v float64) {
	e := kr*r + kg*g + kb*b

	y = 16 + 219.0/255.0*e
	u = 128 + 224.0/255.0*(b-e)/(2*(1-kb))
	v = 128 + 224.0/255.0*(r-e)/(2*(1-kr))

	return y, u, v
}

func clamp(v, lo, hi float64) float64 {
	return min(max(v, lo), hi)
}
