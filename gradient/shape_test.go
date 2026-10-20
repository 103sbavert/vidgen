package gradient

import (
	"math"
	"testing"

	"sbavert/vidgen/config"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-5 }

func TestDirection(t *testing.T) {
	for _, tc := range []struct {
		deg    int
		ux, uy float64
	}{
		{0, 1, 0},
		{90, 0, 1},
		{180, -1, 0},
		{270, 0, -1},
		{45, 0.707107, 0.707107},
		{360, 1, 0},
	} {
		ux, uy := Direction(tc.deg)
		if !near(ux, tc.ux) || !near(uy, tc.uy) {
			t.Errorf("Direction(%d) = %v, %v; want %v, %v", tc.deg, ux, uy, tc.ux, tc.uy)
		}
	}
}

// geq prints the components with %f, so they must already be 6-decimal values.
func TestDirectionIsRoundedToSixDecimals(t *testing.T) {
	for deg := 0; deg <= 360; deg++ {
		ux, uy := Direction(deg)

		for _, v := range []float64{ux, uy} {
			if v != math.Round(v*1e6)/1e6 {
				t.Fatalf("Direction(%d) component %v not rounded", deg, v)
			}
		}
	}
}

func TestRadialShape(t *testing.T) {
	s := RadialShape(200, 100)

	if got := s(100, 50); !near(got, 0) {
		t.Errorf("center = %v, want 0", got)
	}

	if got := s(0, 0); !near(got, 1) {
		t.Errorf("corner = %v, want 1", got)
	}

	if s(100, 0) >= s(0, 0) {
		t.Error("edge midpoint should be closer to center than the corner")
	}
}

func TestLinearShapeEndpoints(t *testing.T) {
	const w, h = 160, 90

	// Start corner of the direction is 1 (first color), opposite corner 0.
	for _, tc := range []struct {
		deg        int
		start, end [2]int
	}{
		{0, [2]int{0, 0}, [2]int{w, 0}},
		{90, [2]int{0, 0}, [2]int{0, h}},
		{180, [2]int{w, 0}, [2]int{0, 0}},
		{270, [2]int{0, h}, [2]int{0, 0}},
		{45, [2]int{0, 0}, [2]int{w, h}},
		{135, [2]int{w, 0}, [2]int{0, h}},
	} {
		s := LinearShape(w, h, tc.deg)

		if got := s(tc.start[0], tc.start[1]); !near(got, 1) {
			t.Errorf("%d°: start corner = %v, want 1", tc.deg, got)
		}

		if got := s(tc.end[0], tc.end[1]); !near(got, 0) {
			t.Errorf("%d°: end corner = %v, want 0", tc.deg, got)
		}
	}
}

func TestLinearShapeAxisAlignedIsConstantAcrossPerpendicular(t *testing.T) {
	horizontal := LinearShape(160, 90, 0)
	vertical := LinearShape(160, 90, 90)

	for _, i := range []int{0, 17, 89} {
		if horizontal(40, i) != horizontal(40, 0) {
			t.Errorf("0°: varies with y at row %d", i)
		}

		if vertical(i, 40) != vertical(0, 40) {
			t.Errorf("90°: varies with x at column %d", i)
		}
	}

	if horizontal(10, 0) <= horizontal(100, 0) {
		t.Error("0°: position should fall from the start edge to the end edge")
	}
}

func TestNewSelectsShape(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.Resolution = "320x180"

	radial, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	cfg.GradientType = config.GradientLinear

	linear, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Different shapes disagree at the center-left of the frame.
	if radial.Shape(0, 90) == linear.Shape(0, 90) {
		t.Error("radial and linear shapes should differ")
	}

	cfg.GradientType = "spiral"
	if _, err := New(cfg); err == nil {
		t.Error("unsupported gradient accepted")
	}

	cfg = config.GetDefaultConfig()
	cfg.Colors = []config.Color{"112233"}

	if _, err := New(cfg); err == nil {
		t.Error("single color accepted")
	}
}
