package config

import (
	"os"
	"path/filepath"
	"slices"
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

func TestOverlayLinesMetadata(t *testing.T) {
	cfg := GetDefaultConfig()

	want := []string{"1920 x 1080 px", "24 fps", "MP4", "libx264", "30s"}
	if got := cfg.OverlayLines(); !slices.Equal(got, want) {
		t.Errorf("before output known: %v, want %v", got, want)
	}

	// The format comes from the output file once it is known.
	out := "clips/test.WEBM"
	cfg.Output = &out

	if got := cfg.OverlayLines()[2]; got != "WEBM" {
		t.Errorf("format = %q, want WEBM", got)
	}

	cfg.Output = nil
	cfg.Codec = "libvpx-vp9"

	if got := cfg.OverlayLines()[2]; got != "WEBM" {
		t.Errorf("format from codec = %q, want WEBM", got)
	}
}

func TestOverlayLinesCustomText(t *testing.T) {
	cfg := GetDefaultConfig()

	text := "hello\nworld"
	cfg.Text = &text

	if got := cfg.OverlayLines(); !slices.Equal(got, []string{"hello", "world"}) {
		t.Errorf("got %v", got)
	}

	empty := ""
	cfg.Text = &empty

	if got := cfg.OverlayLines(); got != nil {
		t.Errorf("empty text should disable the overlay, got %v", got)
	}
}

func TestFontPx(t *testing.T) {
	cfg := GetDefaultConfig() // 1920x1080, metadata

	if got := cfg.FontPx(); got != 38 {
		t.Errorf("metadata 1080p = %d, want 38 (height/28)", got)
	}

	text := "x"
	cfg.Text = &text

	if got := cfg.FontPx(); got != 90 {
		t.Errorf("custom text 1080p = %d, want 90 (height/12)", got)
	}

	cfg.Resolution = "320x240"
	if got := cfg.FontPx(); got != 24 {
		t.Errorf("custom text 240p = %d, want the 24 floor", got)
	}

	cfg.Text = nil
	if got := cfg.FontPx(); got != 14 {
		t.Errorf("metadata 240p = %d, want the 14 floor", got)
	}

	size := 33
	cfg.FontSize = &size

	if got := cfg.FontPx(); got != 33 {
		t.Errorf("explicit size = %d, want 33", got)
	}
}

func TestParseHexKeywordsAndHex(t *testing.T) {
	for color, want := range map[Color][3]uint8{
		"black":  {0x22, 0x22, 0x22},
		"white":  {0xee, 0xee, 0xee},
		"ff8000": {0xff, 0x80, 0x00},
		"FF8000": {0xff, 0x80, 0x00},
	} {
		r, g, b, err := color.ParseHex()
		if err != nil || [3]uint8{r, g, b} != want {
			t.Errorf("%q = %v, %v; want %v", color, [3]uint8{r, g, b}, err, want)
		}
	}

	for _, bad := range []Color{"", "fff", "gggggg", "1234567"} {
		if _, _, _, err := bad.ParseHex(); err == nil {
			t.Errorf("%q accepted", bad)
		}
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
