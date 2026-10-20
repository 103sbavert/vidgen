package gradient

import (
	"math"
	"testing"

	"sbavert/vidgen/config"
)

var defaultColors = []config.Color{"546b41", "99ad7a", "dcccac", "fff8ec"}

func TestBoundaryConstantsDefaultPalette(t *testing.T) {
	// Documented values for N=4.
	for k, want := range []struct{ intercept, target float64 }{
		{0.10, 0.25}, {0.30, 0.50}, {0.60, 0.75},
	} {
		intercept, slope := BoundaryConstants(k+1, 4)

		if math.Abs(intercept-want.intercept) > 1e-12 || math.Abs(intercept+slope-want.target) > 1e-12 {
			t.Errorf("k=%d: intercept=%v slope=%v", k+1, intercept, slope)
		}
	}
}

func TestColorDeltas(t *testing.T) {
	deltas, errs := ColorDeltas(defaultColors)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	want := [][3]int16{{-35, -44, -64}, {-67, -31, -50}, {-69, -66, -57}}
	if len(deltas) != len(want) {
		t.Fatalf("got %d deltas, want %d", len(deltas), len(want))
	}

	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta %d = %v, want %v", i, deltas[i], want[i])
		}
	}
}

func TestNewBandsErrors(t *testing.T) {
	for name, colors := range map[string][]config.Color{
		"none":      nil,
		"one":       {"112233"},
		"bad inner": {"112233", "xyz"},
		"bad outer": {"nothex", "112233"},
	} {
		if _, err := NewBands(colors); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestColorAtEndpoints(t *testing.T) {
	g, err := NewBands(defaultColors)
	if err != nil {
		t.Fatal(err)
	}

	// Center is the innermost color, far edge the outermost, at any progress
	// where all fences have moved clear of the edges.
	if got := g.ColorAt(0, 0); got != [3]float64{255, 248, 236} {
		t.Errorf("center = %v, want FFF8EC", got)
	}

	if got := g.ColorAt(1, 0); got != [3]float64{84, 107, 65} {
		t.Errorf("edge = %v, want 546B41", got)
	}

	// Midway across a fence the blend is exactly half the delta.
	pos := g.Fences[0].At(0)
	got := g.ColorAt(pos, 0)

	for c := range 3 {
		want := float64(g.Base[c]) + float64(g.Deltas[0][c])/2
		if math.Abs(got[c]-want) > 1e-9 {
			t.Errorf("channel %d at fence = %v, want %v", c, got[c], want)
		}
	}
}

func TestColorAtClamped(t *testing.T) {
	// Deltas that overshoot must clamp to 0..255.
	g, err := NewBands([]config.Color{"ffffff", "000000"})
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range []float64{0, 0.25, 0.5, 0.75, 1} {
		for _, v := range g.ColorAt(d, 0.5) {
			if v < 0 || v > 255 {
				t.Fatalf("d=%v: %v out of range", d, v)
			}
		}
	}
}
