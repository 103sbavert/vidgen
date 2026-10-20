package render

import (
	"math"
	"testing"

	"sbavert/vidgen/config"
)

func testConfig(res string) config.VideoConfig {
	cfg := config.GetDefaultConfig()
	cfg.Resolution = config.VideoResolution(res)
	cfg.Duration = 2

	return cfg
}

// referenceRGB evaluates the gradient directly, per pixel, with no tables.
func referenceRGB(r *Radial, x, y, n int) [3]float64 {
	l := r.layout
	cx, cy := float64(l.W)/2, float64(l.H)/2
	d := math.Hypot(float64(x)-cx, float64(y)-cy) / math.Hypot(cx, cy)

	return r.spec.ColorAt(d, float64(n)/float64(r.frames))
}

func TestRadialMatchesReference(t *testing.T) {
	for _, res := range []string{"320x180", "321x181"} {
		r, err := NewRadial(testConfig(res))
		if err != nil {
			t.Fatal(err)
		}

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
				t.Errorf("%s frame %d: max error %.3f levels, want <= 1", res, n, worst)
			}
		}
	}
}

func TestRadialInnerOuterColors(t *testing.T) {
	r, err := NewRadial(testConfig("640x360"))
	if err != nil {
		t.Fatal(err)
	}

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

func BenchmarkRadialRender1080p(b *testing.B) {
	r, err := NewRadial(testConfig("1920x1080"))
	if err != nil {
		b.Fatal(err)
	}

	buf := make([]byte, r.Layout().FrameSize())

	b.SetBytes(int64(len(buf)))
	b.ResetTimer()

	for i := range b.N {
		r.Render(i%r.Frames(), buf)
	}
}

func BenchmarkRadialRender1080pParallel(b *testing.B) {
	r, err := NewRadial(testConfig("1920x1080"))
	if err != nil {
		b.Fatal(err)
	}

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
