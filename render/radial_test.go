package render

import (
	"math"
	"testing"

	"sbavert/vidgen/config"
	"sbavert/vidgen/gradient"
)

const testFrames = 48

func testSpec(t testing.TB, colors ...config.Color) gradient.Radial {
	t.Helper()

	if len(colors) == 0 {
		colors = config.GetDefaultConfig().Colors
	}

	spec, err := gradient.NewRadial(colors)
	if err != nil {
		t.Fatal(err)
	}

	return spec
}

func newTestRadial(t testing.TB, w, h int) *Radial {
	t.Helper()

	r, err := NewRadial(testSpec(t), w, h, testFrames)
	if err != nil {
		t.Fatal(err)
	}

	return r
}

// referenceRGB evaluates the gradient directly, per pixel, with no tables.
func referenceRGB(r *Radial, x, y, n int) [3]float64 {
	l := r.Layout()
	cx, cy := float64(l.W)/2, float64(l.H)/2
	d := math.Hypot(float64(x)-cx, float64(y)-cy) / math.Hypot(cx, cy)

	return r.spec.ColorAt(d, float64(n)/float64(r.frames))
}

func TestRadialMatchesReference(t *testing.T) {
	for _, size := range [][2]int{{320, 180}, {321, 181}} {
		r := newTestRadial(t, size[0], size[1])
		l := r.Layout()
		buf := make([]byte, l.FrameSize())

		for _, n := range []int{0, 1, 17, 30, r.Frames() - 1} {
			r.Render(n, buf)
			yp, up, vp := l.Planes(buf)

			var worst float64

			for y := range l.H {
				for x := range l.W {
					rgb := referenceRGB(r, x, y, n)
					wantY, _, _ := yuv709(rgb[0], rgb[1], rgb[2])
					worst = math.Max(worst, math.Abs(float64(yp[y*l.W+x])-wantY))
				}
			}

			for cy := range l.CH {
				for cx := range l.CW {
					var wantU, wantV float64

					for _, o := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
						rgb := referenceRGB(r, 2*cx+o[0], 2*cy+o[1], n)
						_, u, v := yuv709(rgb[0], rgb[1], rgb[2])
						wantU += u / 4
						wantV += v / 4
					}

					worst = math.Max(worst, math.Abs(float64(up[cy*l.CW+cx])-wantU))
					worst = math.Max(worst, math.Abs(float64(vp[cy*l.CW+cx])-wantV))
				}
			}

			// Rounding to 8 bit plus 12-bit distance quantization.
			if worst > 1.0 {
				t.Errorf("%dx%d frame %d: max error %.3f levels, want <= 1", size[0], size[1], n, worst)
			}
		}
	}
}

func TestRadialInnerOuterColors(t *testing.T) {
	r := newTestRadial(t, 640, 360)
	l := r.Layout()
	buf := make([]byte, l.FrameSize())
	r.Render(0, buf)
	yp, _, _ := l.Planes(buf)

	center := yp[(l.H/2)*l.W+l.W/2]
	corner := yp[0]

	// Innermost color FFF8EC is the lightest, outermost 546B41 the darkest.
	if center <= corner {
		t.Errorf("center luma %d should exceed corner luma %d", center, corner)
	}
}

func TestNewRadialRejectsBadSize(t *testing.T) {
	spec := testSpec(t)

	for _, tc := range []struct{ w, h, frames int }{{0, 10, 1}, {10, 0, 1}, {10, 10, 0}, {-1, 10, 5}} {
		if _, err := NewRadial(spec, tc.w, tc.h, tc.frames); err == nil {
			t.Errorf("NewRadial(%+v) accepted", tc)
		}
	}
}

// Field.Fill is independent of the gradient: a flat table must give flat
// planes, and the chroma average must round-trip an exact value.
func TestFieldFillFlatTable(t *testing.T) {
	l := NewLayout(7, 5) // odd on purpose
	f := newField(l, func(x, y int) float64 { return float64(x+y) / 20 })

	var tbl Table
	for i := range lutSize {
		tbl.set(i, 200, 100, 50)
	}

	buf := make([]byte, l.FrameSize())
	f.Fill(&tbl, buf)

	yp, up, vp := l.Planes(buf)
	wantY, wantU, wantV := yuv709(200, 100, 50)

	for i, v := range yp {
		if v != tbl.y[0] {
			t.Fatalf("luma[%d] = %d, want %d (~%.1f)", i, v, tbl.y[0], wantY)
		}
	}

	for i := range up {
		if math.Abs(float64(up[i])-wantU) > 0.6 || math.Abs(float64(vp[i])-wantV) > 0.6 {
			t.Fatalf("chroma[%d] = %d,%d want ~%.1f,%.1f", i, up[i], vp[i], wantU, wantV)
		}
	}
}

func BenchmarkRadialRender1080p(b *testing.B) {
	r := newTestRadial(b, 1920, 1080)
	buf := make([]byte, r.Layout().FrameSize())

	b.SetBytes(int64(len(buf)))
	b.ResetTimer()

	for i := range b.N {
		r.Render(i%r.Frames(), buf)
	}
}

func BenchmarkRadialRender1080pParallel(b *testing.B) {
	r := newTestRadial(b, 1920, 1080)
	size := r.Layout().FrameSize()

	b.SetBytes(int64(size))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, size)
		i := 0

		for pb.Next() {
			r.Render(i%r.Frames(), buf)
			i++
		}
	})
}
