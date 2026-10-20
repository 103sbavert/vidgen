package render

import (
	"fmt"
	"math"
	"sync"

	"sbavert/vidgen/gradient"
)

// bandTable is a Table plus the scratch space its builder needs, pooled
// together so frames do not allocate.
type bandTable struct {
	Table

	acc  [3][lutSize]float64
	step [3][lutSize + 1]float64
}

// build tabulates the gradient at progress p (0→1).
//
// For each fence only the smoothstep transition band touches individual
// entries (acc); everything beyond the band gets the full delta, applied via
// a running sum over step. Total cost is O(lutSize + fences*bandWidth).
func (t *bandTable) build(g *gradient.Bands, p float64) {
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

// Banded renders any gradient.Gradient: banded colors laid out by a shape
// (radial, linear, ...). It is safe for concurrent use: the position field is
// immutable and the per-frame tables come from a pool.
type Banded struct {
	field  *Field
	bands  gradient.Bands
	frames int
	pool   sync.Pool
}

// NewBanded prepares a renderer for a w x h clip of frames frames, evaluating
// g.Shape for every pixel up front. g.Shape must have been built for w x h.
func NewBanded(g gradient.Gradient, w, h, frames int) (*Banded, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid frame size %dx%d", w, h)
	}

	if frames <= 0 {
		return nil, fmt.Errorf("invalid frame count %d", frames)
	}

	if g.Shape == nil {
		return nil, fmt.Errorf("gradient has no shape")
	}

	r := &Banded{
		bands:  g.Bands,
		frames: frames,
		field:  newField(NewLayout(w, h), g.Shape),
	}
	r.pool.New = func() any { return new(bandTable) }

	return r, nil
}

// Layout returns the frame layout produced by Render.
func (r *Banded) Layout() Layout { return r.field.layout }

// Frames is the total number of frames in the clip.
func (r *Banded) Frames() int { return r.frames }

// Render fills buf (len >= Layout().FrameSize()) with frame n.
func (r *Banded) Render(n int, buf []byte) {
	t := r.pool.Get().(*bandTable)
	defer r.pool.Put(t)

	t.build(&r.bands, float64(n)/float64(r.frames))
	r.field.Fill(&t.Table, buf)
}
