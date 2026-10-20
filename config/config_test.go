package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigValidates(t *testing.T) {
	if err := GetDefaultConfig().Validate(); err != nil {
		t.Fatal(err)
	}
}

// The constants and the `oneof` validation tags are written separately; this
// keeps them from drifting apart.
func TestEngineAndGradientConstantsMatchValidation(t *testing.T) {
	for _, engine := range []string{EngineNative, EngineGeq} {
		cfg := GetDefaultConfig()
		cfg.Engine = engine

		if err := cfg.Validate(); err != nil {
			t.Errorf("engine %q rejected: %v", engine, err)
		}
	}

	cfg := GetDefaultConfig()
	cfg.GradientType = GradientRadial

	if err := cfg.Validate(); err != nil {
		t.Errorf("gradient %q rejected: %v", GradientRadial, err)
	}

	cfg.Engine = "bogus"
	if cfg.Validate() == nil {
		t.Error("unknown engine accepted")
	}
}

func TestValidateRejectsBadParallelism(t *testing.T) {
	cfg := GetDefaultConfig()
	cfg.Workers = -1

	if cfg.Validate() == nil {
		t.Error("negative workers accepted")
	}

	cfg = GetDefaultConfig()
	cfg.Segments = 65

	if cfg.Validate() == nil {
		t.Error("65 segments accepted")
	}
}

func TestLookupCodec(t *testing.T) {
	if info, ok := LookupCodec("libvpx-vp9"); !ok || info.Ext != ExtWebm || !info.SplitByDefault {
		t.Errorf("libvpx-vp9 = %+v, %v", info, ok)
	}

	if info, ok := LookupCodec("libx264"); !ok || info.Ext != ExtMp4 || info.SplitByDefault {
		t.Errorf("libx264 = %+v, %v", info, ok)
	}

	if _, ok := LookupCodec("nope"); ok {
		t.Error("unknown codec found")
	}
}

func TestResolutionSize(t *testing.T) {
	w, h, err := VideoResolution("1920x1080").Size()
	if err != nil || w != 1920 || h != 1080 {
		t.Errorf("got %dx%d, %v", w, h, err)
	}

	if _, _, err := VideoResolution("abc").Size(); err == nil {
		t.Error("bad resolution accepted")
	}
}

// Regression: CalcOutputPath used a value receiver, so the auto-picked path
// was dropped and Output stayed nil.
func TestCalcOutputPathSetsOutput(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := GetDefaultConfig()
	if err := cfg.CalcOutputPath(); err != nil {
		t.Fatal(err)
	}

	if cfg.Output == nil || *cfg.Output != "radial_1920x1080_24fps_2097152bps_30s.mp4" {
		t.Fatalf("Output = %v", cfg.Output)
	}

	// Taken auto-names get a numbered suffix.
	if err := os.WriteFile(*cfg.Output, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	next := GetDefaultConfig()
	if err := next.CalcOutputPath(); err != nil {
		t.Fatal(err)
	}

	if *next.Output != "radial_1920x1080_24fps_2097152bps_30s_1.mp4" {
		t.Errorf("second Output = %q", *next.Output)
	}
}

func TestCalcOutputPathRefusesOverwrite(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "out.mp4")
	if err := os.WriteFile(existing, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := GetDefaultConfig()
	cfg.Output = &existing

	if cfg.CalcOutputPath() == nil {
		t.Error("existing explicit output accepted")
	}
}
