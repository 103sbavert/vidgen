package render

import (
	"fmt"
	"math"
	"testing"

	"sbavert/vidgen/config"
	"sbavert/vidgen/gradient"
)

const testFrames = 48

// testGradient builds the gradient the real pipeline would for these settings.
func testGradient(t testing.TB, gradientType string, angle, w, h int) gradient.Gradient {
	t.Helper()

	cfg := config.GetDefaultConfig()
	cfg.GradientType = gradientType
	cfg.LinearAngle = angle
	cfg.Resolution = config.VideoResolution(fmt.Sprintf("%dx%d", w, h))

	g, err := gradient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return g
}

func newTestBanded(t testing.TB, gradientType string, angle, w, h int) (*Banded, gradient.Gradient) {
	t.Helper()

	g := testGradient(t, gradientType, angle, w, h)

	r, err := NewBanded(g, w, h, testFrames)
	if err != nil {
		t.Fatal(err)
	}

	return r, g
}

func TestBandedMatchesReference(t *testing.T) {
	for _, tc := range []struct {
		name         string
		gradientType string
		angle        int
		w, h         int
	}{
		{"radial", config.GradientRadial, 0, 320, 180},
		{"radial odd size", config.GradientRadial, 0, 321, 181},
		{"linear 0", config.GradientLinear, 0, 320, 180},
		{"linear 90", config.GradientLinear, 90, 320, 180},
		{"linear 37 odd size", config.GradientLinear, 37, 321, 181},
		{"linear 225", config.GradientLinear, 225, 320, 180},
	} {
		r, g := newTestBanded(t, tc.gradientType, tc.angle, tc.w, tc.h)
		l := r.Layout()
		buf := make([]byte, l.FrameSize())

		for _, n := range []int{0, 1, 17, 30, r.Frames() - 1} {
			r.Render(n, buf)
			yp, up, vp := l.Planes(buf)
			p := float64(n) / float64(r.Frames())

			var worst float64

			ref := func(x, y int) [3]float64 { return g.Bands.ColorAt(g.Shape(x, y), p) }

			for y := range l.H {
				for x := range l.W {
					rgb := ref(x, y)
					wantY, _, _ := yuv709(rgb[0], rgb[1], rgb[2])
					worst = math.Max(worst, math.Abs(float64(yp[y*l.W+x])-wantY))
				}
			}

			for cy := range l.CH {
				for cx := range l.CW {
					var wantU, wantV float64

					for _, o := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
						rgb := ref(2*cx+o[0], 2*cy+o[1])
						_, u, v := yuv709(rgb[0], rgb[1], rgb[2])
						wantU += u / 4
						wantV += v / 4
					}

					worst = math.Max(worst, math.Abs(float64(up[cy*l.CW+cx])-wantU))
					worst = math.Max(worst, math.Abs(float64(vp[cy*l.CW+cx])-wantV))
				}
			}

			// Rounding to 8 bit plus 12-bit position quantization.
			if worst > 1.0 {
				t.Errorf("%s frame %d: max error %.3f levels, want <= 1", tc.name, n, worst)
			}
		}
	}
}

// Radial: lightest color (innermost) in the middle, darkest (outermost) in the
// corner.
func TestBandedRadialColors(t *testing.T) {
	r, _ := newTestBanded(t, config.GradientRadial, 0, 640, 360)
	l := r.Layout()
	buf := make([]byte, l.FrameSize())
	r.Render(0, buf)
	yp, _, _ := l.Planes(buf)

	if center, corner := yp[(l.H/2)*l.W+l.W/2], yp[0]; center <= corner {
		t.Errorf("center luma %d should exceed corner luma %d", center, corner)
	}
}

// Linear: the first (darkest) color starts at the origin edge, the last
// (lightest) at the far edge; the angle picks the edge.
func TestBandedLinearOrientation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		angle int
		// darker pixel, lighter pixel
		dark, light [2]int
	}{
		{"0 left to right", 0, [2]int{2, 180}, [2]int{637, 180}},
		{"90 top to bottom", 90, [2]int{320, 2}, [2]int{320, 357}},
		{"180 right to left", 180, [2]int{637, 180}, [2]int{2, 180}},
		{"270 bottom to top", 270, [2]int{320, 357}, [2]int{320, 2}},
	} {
		r, _ := newTestBanded(t, config.GradientLinear, tc.angle, 640, 360)
		l := r.Layout()
		buf := make([]byte, l.FrameSize())
		r.Render(0, buf)
		yp, _, _ := l.Planes(buf)

		dark := yp[tc.dark[1]*l.W+tc.dark[0]]
		light := yp[tc.light[1]*l.W+tc.light[0]]

		if dark >= light {
			t.Errorf("%s: luma at start %d should be below luma at end %d", tc.name, dark, light)
		}
	}
}

// A 0 degree linear gradient varies only with x; 90 only with y.
func TestBandedLinearAxisAligned(t *testing.T) {
	for _, angle := range []int{0, 90} {
		r, _ := newTestBanded(t, config.GradientLinear, angle, 400, 300)
		l := r.Layout()
		buf := make([]byte, l.FrameSize())
		r.Render(20, buf)
		yp, _, _ := l.Planes(buf)

		// Two pixels that differ only along the axis perpendicular to the
		// gradient must have the same luma.
		for _, i := range []int{10, 150, 290} {
			ax, ay, bx, by := i, 50, i, 250
			if angle == 90 {
				ax, ay, bx, by = 50, i, 350, i
			}

			if a, b := yp[ay*l.W+ax], yp[by*l.W+bx]; a != b {
				t.Errorf("angle %d: luma differs across the constant axis: %d vs %d", angle, a, b)
			}
		}
	}
}

func TestNewBandedRejectsBadInput(t *testing.T) {
	g := testGradient(t, config.GradientRadial, 0, 320, 180)

	for _, tc := range []struct{ w, h, frames int }{{0, 10, 1}, {10, 0, 1}, {10, 10, 0}, {-1, 10, 5}} {
		if _, err := NewBanded(g, tc.w, tc.h, tc.frames); err == nil {
			t.Errorf("NewBanded(%+v) accepted", tc)
		}
	}

	if _, err := NewBanded(gradient.Gradient{Bands: g.Bands}, 10, 10, 5); err == nil {
		t.Error("nil shape accepted")
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

func BenchmarkBandedRender1080p(b *testing.B) {
	for _, gt := range []string{config.GradientRadial, config.GradientLinear} {
		b.Run(gt, func(b *testing.B) {
			r, _ := newTestBanded(b, gt, 30, 1920, 1080)
			buf := make([]byte, r.Layout().FrameSize())

			b.SetBytes(int64(len(buf)))
			b.ResetTimer()

			for i := range b.N {
				r.Render(i%r.Frames(), buf)
			}
		})
	}
}

func BenchmarkBandedRender1080pParallel(b *testing.B) {
	r, _ := newTestBanded(b, config.GradientRadial, 0, 1920, 1080)
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
