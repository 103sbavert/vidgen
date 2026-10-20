package geq

import (
	"os"
	"strings"
	"testing"

	"sbavert/vidgen/config"
)

// The radial golden files were produced by the original (pre-gradient-package)
// implementation, so they pin the exact expression ffmpeg receives. The
// linear one pins the current expression to catch accidental changes; its
// correctness is checked end to end against the native renderer in package
// ffmpeg.
func TestBuildGolden(t *testing.T) {
	threeColors := config.GetDefaultConfig()
	threeColors.Colors = []config.Color{"112233", "445566", "778899"}
	threeColors.Duration = 10

	linear := config.GetDefaultConfig()
	linear.GradientType = config.GradientLinear
	linear.LinearAngle = 30

	for _, tc := range []struct {
		name   string
		cfg    config.VideoConfig
		golden string
	}{
		{"default", config.GetDefaultConfig(), "testdata/radial_default.golden"},
		{"three colors", threeColors, "testdata/radial_3color.golden"},
		{"linear 30", linear, "testdata/linear_30.golden"},
	} {
		want, err := os.ReadFile(tc.golden)
		if err != nil {
			t.Fatal(err)
		}

		got, err := Build(tc.cfg)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		if got != string(want) {
			t.Errorf("%s: expression changed\n got: %s\nwant: %s", tc.name, got, want)
		}
	}
}

func TestBuildErrors(t *testing.T) {
	cfg := config.GetDefaultConfig()

	cfg.Colors = nil
	if _, err := Build(cfg); err == nil {
		t.Error("no colors: want error")
	}

	cfg.Colors = []config.Color{"112233"}
	if _, err := Build(cfg); err == nil {
		t.Error("one color: want error")
	}

	cfg.Colors = []config.Color{"112233", "zzzzzz"}
	if _, err := Build(cfg); err == nil {
		t.Error("bad color: want error")
	}

	cfg = config.GetDefaultConfig()
	cfg.GradientType = "spiral"

	if _, err := Build(cfg); err == nil {
		t.Error("unsupported gradient: want error")
	}
}

// Negative direction components must stay valid geq: parenthesized, never a
// bare `*-0.5`.
func TestLinearShapeExprParenthesizesConstants(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.GradientType = config.GradientLinear
	cfg.LinearAngle = 225

	expr, err := shapeExpr(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(expr, "*(-0.707107)") {
		t.Errorf("expected parenthesized negative constants in %q", expr)
	}

	if strings.Contains(expr, "*-") {
		t.Errorf("bare negative operand in %q", expr)
	}
}
