package geq

import (
	"os"
	"testing"

	"sbavert/vidgen/config"
)

// The golden files were produced by the original (pre-gradient-package)
// implementation, so they pin the exact expression ffmpeg receives.
func TestBuildRadialGeqGolden(t *testing.T) {
	threeColors := config.GetDefaultConfig()
	threeColors.Colors = []config.Color{"112233", "445566", "778899"}
	threeColors.Duration = 10

	for _, tc := range []struct {
		name   string
		cfg    config.VideoConfig
		golden string
	}{
		{"default", config.GetDefaultConfig(), "testdata/radial_default.golden"},
		{"three colors", threeColors, "testdata/radial_3color.golden"},
	} {
		want, err := os.ReadFile(tc.golden)
		if err != nil {
			t.Fatal(err)
		}

		got, err := BuildRadialGeq(tc.cfg)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		if got != string(want) {
			t.Errorf("%s: expression changed\n got: %s\nwant: %s", tc.name, got, want)
		}
	}
}

func TestBuildRadialGeqErrors(t *testing.T) {
	cfg := config.GetDefaultConfig()

	cfg.Colors = nil
	if _, err := BuildRadialGeq(cfg); err == nil {
		t.Error("no colors: want error")
	}

	cfg.Colors = []config.Color{"112233"}
	if _, err := BuildRadialGeq(cfg); err == nil {
		t.Error("one color: want error")
	}

	cfg.Colors = []config.Color{"112233", "zzzzzz"}
	if _, err := BuildRadialGeq(cfg); err == nil {
		t.Error("bad color: want error")
	}
}
