package render

import (
	"fmt"
	"math"
	"sync"

	"sbavert/vidgen/gradient"
)

// radialTable is a Table plus the scratch space its builder needs, pooled
// together so frames do not allocate.
type radialTable struct {
	Table

	acc  [3][lutSize]float64
	step [3][lutSize + 1]float64
}

// build tabulates the gradient at progress p (0→1).
//
// For each fence only the smoothstep transition band touches individual
// entries (acc); everything beyond the band gets the full delta, applied via
// a running sum over step. Total cost is O(lutSize + fences*bandWidth).
func (t *radialTable) build(g *gradient.Radial, p float64) {
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
			s := gradient.Smoothstep(float64(i)/scale, pos, sigma)

			for c := range 3 {
				t.acc[c][i] += float64(delta[c]) * s
			}
		}

		if hiRaw+1 < lutSize {
			at := max(0, hiRaw+1)

			for c := range 3 {
				t.step[c][at] += float64(delta[c])
			}
		}
	}

	var run [3]float64

	for i := range lutSize {
		var rgb [3]float64

		for c := range 3 {
			run[c] += t.step[c][i]
			rgb[c] = clamp(float64(g.Base[c])+run[c]+t.acc[c][i], 0, 255)
		}

		t.set(i, rgb[0], rgb[1], rgb[2])
	}
}

// Radial renders the animated radial gradient described by a gradient.Radial.
// It is safe for concurrent use: the distance field is immutable and the
// per-frame tables come from a pool.
type Radial struct {
	field  *Field
	spec   gradient.Radial
	frames int
	pool   sync.Pool
}

// NewRadial prepares a renderer for a w x h clip of frames frames, computing
// the distance field up front.
func NewRadial(spec gradient.Radial, w, h, frames int) (*Radial, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid frame size %dx%d", w, h)
	}

	if frames <= 0 {
		return nil, fmt.Errorf("invalid frame count %d", frames)
	}

	// Same definition as the geq expression: integer pixel coordinates,
	// distance to (W/2, H/2) normalized by the half-diagonal.
	cx, cy := float64(w)/2, float64(h)/2
	norm := 1 / math.Hypot(cx, cy)

	r := &Radial{
		spec:   spec,
		frames: frames,
		field: newField(NewLayout(w, h), func(x, y int) float64 {
			return math.Hypot(float64(x)-cx, float64(y)-cy) * norm
		}),
	}
	r.pool.New = func() any { return new(radialTable) }

	return r, nil
}

// Layout returns the frame layout produced by Render.
func (r *Radial) Layout() Layout { return r.field.layout }

// Frames is the total number of frames in the clip.
func (r *Radial) Frames() int { return r.frames }

// Render fills buf (len >= Layout().FrameSize()) with frame n.
func (r *Radial) Render(n int, buf []byte) {
	t := r.pool.Get().(*radialTable)
	defer r.pool.Put(t)

	t.build(&r.spec, float64(n)/float64(r.frames))
	r.field.Fill(&t.Table, buf)
}
