package render

import "sbavert/vidgen/caption"

// Overlay blends one solid color through a coverage mask into yuv420p frames.
// It is built once: the mask is aligned to the chroma grid and chroma coverage
// is precomputed, so applying it costs only the pixels the ink touches.
type Overlay struct {
	layout Layout

	// Even-aligned rectangle (frame coordinates) covering the mask.
	x0, y0, w, h int

	alpha  []uint8 // w*h luma coverage
	chroma []uint8 // (w/2)*(h/2) coverage averaged over each 2x2 block

	y, u, v uint8 // text color, BT.709 limited range
}

// NewOverlay prepares the overlay of mask in color rgb (full-range 0..255) for
// frames of layout l. It returns nil when the mask draws nothing.
func NewOverlay(l Layout, m *caption.Mask, rgb [3]uint8) *Overlay {
	if m.Empty() {
		return nil
	}

	o := &Overlay{layout: l}

	o.x0, o.y0 = m.X&^1, m.Y&^1
	o.w = (m.X+m.W+1)&^1 - o.x0
	o.h = (m.Y+m.H+1)&^1 - o.y0

	o.alpha = make([]uint8, o.w*o.h)
	for row := range m.H {
		dst := o.alpha[(m.Y-o.y0+row)*o.w+(m.X-o.x0):]
		copy(dst[:m.W], m.Alpha[row*m.W:(row+1)*m.W])
	}

	cw := o.w / 2
	o.chroma = make([]uint8, cw*(o.h/2))

	for cy := range o.h / 2 {
		top := o.alpha[2*cy*o.w : (2*cy+1)*o.w]
		bot := o.alpha[(2*cy+1)*o.w : (2*cy+2)*o.w]

		for cx := range cw {
			sum := int(top[2*cx]) + int(top[2*cx+1]) + int(bot[2*cx]) + int(bot[2*cx+1])
			o.chroma[cy*cw+cx] = uint8((sum + 2) / 4)
		}
	}

	y, u, v := yuv709(float64(rgb[0]), float64(rgb[1]), float64(rgb[2]))
	o.y, o.u, o.v = uint8(y+0.5), uint8(u+0.5), uint8(v+0.5)

	return o
}

// blend returns dst moved toward src by coverage a (0..255), rounded.
func blend(dst, src, a uint8) uint8 {
	return uint8((int(dst)*(255-int(a)) + int(src)*int(a) + 127) / 255)
}

// Apply composites the overlay onto one frame in buf (len >= FrameSize).
func (o *Overlay) Apply(buf []byte) {
	l := o.layout
	yPlane, uPlane, vPlane := l.Planes(buf)

	for row := 0; row < o.h && o.y0+row < l.H; row++ {
		cover := o.alpha[row*o.w : (row+1)*o.w]
		dst := yPlane[(o.y0+row)*l.W : (o.y0+row+1)*l.W]

		for col, a := range cover {
			if x := o.x0 + col; a != 0 && x < l.W {
				dst[x] = blend(dst[x], o.y, a)
			}
		}
	}

	cw, cx0, cy0 := o.w/2, o.x0/2, o.y0/2

	for row := 0; row < o.h/2 && cy0+row < l.CH; row++ {
		cover := o.chroma[row*cw : (row+1)*cw]
		du := uPlane[(cy0+row)*l.CW : (cy0+row+1)*l.CW]
		dv := vPlane[(cy0+row)*l.CW : (cy0+row+1)*l.CW]

		for col, a := range cover {
			if x := cx0 + col; a != 0 && x < l.CW {
				du[x] = blend(du[x], o.u, a)
				dv[x] = blend(dv[x], o.v, a)
			}
		}
	}
}

// withOverlay composites an Overlay on every frame of another Source.
type withOverlay struct {
	Source
	overlay *Overlay
}

// WithOverlay returns src with o drawn on every frame. A nil overlay returns
// src unchanged.
func WithOverlay(src Source, o *Overlay) Source {
	if o == nil {
		return src
	}

	return withOverlay{Source: src, overlay: o}
}

func (s withOverlay) Render(n int, buf []byte) {
	s.Source.Render(n, buf)
	s.overlay.Apply(buf)
}
