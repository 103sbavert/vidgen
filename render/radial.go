package render

import (
	"math"
	"runtime"
	"sync"

	"sbavert/vidgen/config"
	"sbavert/vidgen/gradient"
)

const (
	// lutBits sets the distance quantization: a frame's colors are tabulated
	// at lutSize distances. 4096 steps keep the error far below one 8-bit
	// level even on the steepest ring transition.
	lutBits = 12
	lutSize = 1 << lutBits
	lutMask = lutSize - 1
)

// lut maps a quantized distance index to final yuv420p code values.
// Chroma is kept with two extra fractional bits so that averaging four
// samples (2x2 luma neighbourhood) rounds exactly once, at the end.
type lut struct {
	y [lutSize]uint8
	u [lutSize]uint16
	v [lutSize]uint16

	// scratch used by build; kept here so it is pooled with the table.
	acc  [3][lutSize]float64
	step [3][lutSize + 1]float64
}

// build tabulates the gradient at progress p (0→1).
//
// For each fence only the smoothstep transition band touches individual
// entries (acc); everything beyond the band gets the full delta, applied via
// a running sum over step. Total cost is O(lutSize + fences*bandWidth).
func (t *lut) build(g *gradient.Radial, p float64) {
	t.acc = [3][lutSize]float64{}
	t.step = [3][lutSize + 1]float64{}

	const scale = lutSize - 1
	sigma := g.Sigma

	for k, fence := range g.Fences {
		pos := fence.At(p)
		delta := g.Deltas[k]

		lo := max(0, int(math.Ceil((pos-sigma)*scale)))
		hiRaw := int(math.Floor((pos + sigma) * scale))
		hi := min(lutSize-1, hiRaw)

		for i := lo; i <= hi; i++ {
			d := float64(i) / scale
			s := clamp((d-pos+sigma)/(2*sigma), 0, 1)
			s = s * s * (3 - 2*s)

			t.acc[0][i] += float64(delta[0]) * s
			t.acc[1][i] += float64(delta[1]) * s
			t.acc[2][i] += float64(delta[2]) * s
		}

		if hiRaw+1 < lutSize {
			at := max(0, hiRaw+1)
			t.step[0][at] += float64(delta[0])
			t.step[1][at] += float64(delta[1])
			t.step[2][at] += float64(delta[2])
		}
	}

	var run [3]float64

	for i := range lutSize {
		var rgb [3]float64

		for c := range 3 {
			run[c] += t.step[c][i]
			rgb[c] = clamp(float64(g.Base[c])+run[c]+t.acc[c][i], 0, 255)
		}

		y, u, v := yuv709(rgb[0], rgb[1], rgb[2])

		t.y[i] = uint8(y + 0.5)
		t.u[i] = uint16(u*4 + 0.5)
		t.v[i] = uint16(v*4 + 0.5)
	}
}

// Radial renders the animated radial gradient. It is safe for concurrent use:
// the distance map is immutable and per-frame tables come from a pool.
type Radial struct {
	layout Layout
	spec   gradient.Radial
	frames int

	// idx holds, for every pixel of the even-padded frame, the quantized
	// normalized distance to the center. Computed once; reused every frame.
	idx []uint16

	pool sync.Pool
}

// NewRadial prepares a renderer for cfg: parses the palette and precomputes
// the distance map.
func NewRadial(cfg config.VideoConfig) (*Radial, error) {
	spec, err := gradient.NewRadial(cfg.Colors)
	if err != nil {
		return nil, err
	}

	w, h, err := cfg.Resolution.Size()
	if err != nil {
		return nil, err
	}

	r := &Radial{
		layout: NewLayout(w, h),
		spec:   spec,
		frames: cfg.Framerate * cfg.Duration,
	}
	r.pool.New = func() any { return new(lut) }
	r.buildDistanceMap()

	return r, nil
}

// Layout returns the frame layout produced by Render.
func (r *Radial) Layout() Layout { return r.layout }

// Frames is the total number of frames in the clip (fps * duration).
func (r *Radial) Frames() int { return r.frames }

func (r *Radial) buildDistanceMap() {
	l := r.layout
	r.idx = make([]uint16, l.PW*l.PH)

	// Same definition as the geq expression: integer pixel coordinates,
	// distance to (W/2, H/2) normalized by the half-diagonal.
	cx, cy := float64(l.W)/2, float64(l.H)/2
	norm := 1 / math.Hypot(cx, cy)

	workers := runtime.NumCPU()
	rowsPer := (l.PH + workers - 1) / workers

	var wg sync.WaitGroup

	for start := 0; start < l.PH; start += rowsPer {
		end := min(start+rowsPer, l.PH)

		wg.Add(1)

		go func() {
			defer wg.Done()

			for y := start; y < end; y++ {
				dy := float64(y) - cy
				row := r.idx[y*l.PW : (y+1)*l.PW]

				for x := range row {
					d := math.Hypot(float64(x)-cx, dy) * norm
					row[x] = uint16(min(lutSize-1, int(d*(lutSize-1)+0.5)))
				}
			}
		}()
	}

	wg.Wait()
}

// Render fills buf (len >= Layout().FrameSize()) with frame n.
func (r *Radial) Render(n int, buf []byte) {
	t := r.pool.Get().(*lut)
	defer r.pool.Put(t)

	t.build(&r.spec, float64(n)/float64(r.frames))

	l := r.layout
	yPlane, uPlane, vPlane := l.Planes(buf)

	for y := range l.H {
		src := r.idx[y*l.PW : y*l.PW+l.W]
		dst := yPlane[y*l.W : (y+1)*l.W]

		for x, d := range src {
			dst[x] = t.y[d&lutMask]
		}
	}

	for cy := range l.CH {
		top := r.idx[2*cy*l.PW : (2*cy+1)*l.PW]
		bot := r.idx[(2*cy+1)*l.PW : (2*cy+2)*l.PW]
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
