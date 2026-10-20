// Package render generates raw video frames natively in Go. Frames are
// planar yuv420p (limited range, BT.709) and are meant to be piped straight
// into an encoder, which replaces the slow per-pixel ffmpeg geq filter.
package render

// Layout describes the memory layout of one planar yuv420p frame:
//
//	[ Y: W*H ][ U: CW*CH ][ V: CW*CH ]
//
// PW/PH are W/H rounded up to even, so every chroma sample has a full 2x2
// luma neighbourhood to average over (odd sizes replicate the edge).
type Layout struct {
	W, H   int
	CW, CH int
	PW, PH int
}

// NewLayout returns the yuv420p layout for a w x h frame.
func NewLayout(w, h int) Layout {
	cw, ch := (w+1)/2, (h+1)/2

	return Layout{W: w, H: h, CW: cw, CH: ch, PW: cw * 2, PH: ch * 2}
}

// FrameSize is the byte size of one frame.
func (l Layout) FrameSize() int {
	return l.W*l.H + 2*l.CW*l.CH
}

// Planes slices buf (len >= FrameSize) into its Y, U and V planes.
func (l Layout) Planes(buf []byte) (y, u, v []byte) {
	ySize := l.W * l.H
	cSize := l.CW * l.CH

	y = buf[:ySize:ySize]
	u = buf[ySize : ySize+cSize : ySize+cSize]
	v = buf[ySize+cSize : ySize+2*cSize : ySize+2*cSize]

	return y, u, v
}
