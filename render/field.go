package render

import (
	"runtime"
	"sync"
)

// Field holds, for every pixel of the even-padded frame, a quantized scalar
// in [0, lutSize) such as the normalized distance to the center. It depends
// only on geometry, so it is computed once and reused for every frame; each
// frame then needs just a Table and one lookup per pixel.
//
// Any gradient whose color is a function of one per-pixel scalar and time
// can reuse Field.Fill and the Stream pipeline by supplying its own scalar
// function and Table builder.
type Field struct {
	layout Layout
	idx    []uint16
}

// newField evaluates scalar(x, y) for every pixel of the padded frame. scalar
// returns a value in [0, 1]; it is quantized to lutSize steps. It must be safe
// for concurrent calls.
func newField(l Layout, scalar func(x, y int) float64) *Field {
	f := &Field{layout: l, idx: make([]uint16, l.PW*l.PH)}

	workers := runtime.NumCPU()
	rowsPer := (l.PH + workers - 1) / workers

	var wg sync.WaitGroup

	for start := 0; start < l.PH; start += rowsPer {
		end := min(start+rowsPer, l.PH)

		wg.Go(func() {
			for y := start; y < end; y++ {
				row := f.idx[y*l.PW : (y+1)*l.PW]

				for x := range row {
					row[x] = uint16(min(lutSize-1, max(0, int(scalar(x, y)*(lutSize-1)+0.5))))
				}
			}
		})
	}

	wg.Wait()

	return f
}

// Fill writes one frame into buf (len >= Layout.FrameSize) by looking every
// pixel up in t. Chroma samples average the table over their 2x2 luma block.
func (f *Field) Fill(t *Table, buf []byte) {
	l := f.layout
	yPlane, uPlane, vPlane := l.Planes(buf)

	for y := range l.H {
		src := f.idx[y*l.PW : y*l.PW+l.W]
		dst := yPlane[y*l.W : (y+1)*l.W]

		for x, d := range src {
			dst[x] = t.y[d&lutMask]
		}
	}

	for cy := range l.CH {
		top := f.idx[2*cy*l.PW : (2*cy+1)*l.PW]
		bot := f.idx[(2*cy+1)*l.PW : (2*cy+2)*l.PW]
		du := uPlane[cy*l.CW : (cy+1)*l.CW]
		dv := vPlane[cy*l.CW : (cy+1)*l.CW]

		for cx := range l.CW {
			a, b := top[2*cx]&lutMask, top[2*cx+1]&lutMask
			c, d := bot[2*cx]&lutMask, bot[2*cx+1]&lutMask

			du[cx] = uint8((t.u[a] + t.u[b] + t.u[c] + t.u[d] + 8) >> 4)
			dv[cx] = uint8((t.v[a] + t.v[b] + t.v[c] + t.v[d] + 8) >> 4)
		}
	}
}
