package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"sbavert/vidgen/config"
)

// endToEndConfig is a tiny clip on the built-in mpeg4 encoder (available in
// every ffmpeg build) writing into a fresh temp dir.
func endToEndConfig(t *testing.T) (config.VideoConfig, string) {
	t.Helper()

	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}

	out := filepath.Join(t.TempDir(), "clip.mp4")

	cfg := config.GetDefaultConfig()
	cfg.Resolution = "320x240"
	cfg.Duration = 2
	cfg.Codec = "mpeg4"
	cfg.Output = &out

	return cfg, out
}

func packetCount(t *testing.T, path string) int {
	t.Helper()

	raw, err := exec.Command("ffprobe", "-v", "error", "-count_packets", "-select_streams", "v",
		"-show_entries", "stream=nb_read_packets", "-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe %s: %v", path, err)
	}

	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("ffprobe output %q: %v", raw, err)
	}

	return n
}

func assertOnlyOutput(t *testing.T, dir, name string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if e.Name() != name {
			t.Errorf("leftover file %q next to output", e.Name())
		}
	}
}

func TestEncodeNative(t *testing.T) {
	for _, segments := range []int{1, 3} {
		cfg, out := endToEndConfig(t)
		cfg.Segments = segments

		// OnFrame may run concurrently with several segments.
		var frames atomic.Int64

		opts := Options{OnFrame: func() { frames.Add(1) }}

		if err := Encode(context.Background(), cfg, opts); err != nil {
			t.Fatalf("segments=%d: %v", segments, err)
		}

		if got := packetCount(t, out); got != cfg.TotalFrames() {
			t.Errorf("segments=%d: %d packets, want %d", segments, got, cfg.TotalFrames())
		}

		if got := int(frames.Load()); got != cfg.TotalFrames() {
			t.Errorf("segments=%d: OnFrame called %d times, want %d", segments, got, cfg.TotalFrames())
		}

		assertOnlyOutput(t, filepath.Dir(out), "clip.mp4")
	}
}

func TestEncodeGeq(t *testing.T) {
	cfg, out := endToEndConfig(t)
	cfg.Engine = config.EngineGeq
	cfg.Duration = 1

	if err := Encode(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}

	if got := packetCount(t, out); got != cfg.TotalFrames() {
		t.Errorf("%d packets, want %d", got, cfg.TotalFrames())
	}

	assertOnlyOutput(t, filepath.Dir(out), "clip.mp4")
}

// psnr compares two clips as RGB with ffmpeg and returns the average PSNR in
// dB over all frames.
func psnr(t *testing.T, a, b string) float64 {
	t.Helper()

	return psnrFiltered(t, a, b, "")
}

// psnrFiltered is psnr after applying an extra filter (e.g. a crop) to both
// clips; pre may be empty.
func psnrFiltered(t *testing.T, a, b, pre string) float64 {
	t.Helper()

	if pre != "" {
		pre += ","
	}

	out, err := exec.Command("ffmpeg", "-hide_banner", "-i", a, "-i", b,
		"-filter_complex", "[0:v]"+pre+"format=gbrp[a];[1:v]"+pre+"format=gbrp[b];[a][b]psnr",
		"-f", "null", "-").CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg psnr: %v\n%s", err, out)
	}

	_, tail, ok := strings.Cut(string(out), "average:")
	if !ok {
		t.Fatalf("no PSNR in output:\n%s", out)
	}

	value, _, _ := strings.Cut(tail, " ")

	db, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("parse PSNR %q: %v", value, err)
	}

	return db
}

// The geq engine is the reference: the native renderer must produce the same
// picture for every gradient type and angle. A wrong orientation, flipped
// color order or wrong normalization drops PSNR far below the threshold.
func TestNativeMatchesGeqReference(t *testing.T) {
	// Matching clips measure ~37 dB here (4:2:0 + lossy encode noise on a
	// tiny frame); mismatched ones ~10 dB.
	const minPSNR = 30.0

	for _, tc := range []struct {
		name         string
		gradientType string
		angle        int
	}{
		{"radial", config.GradientRadial, 0},
		{"linear 0", config.GradientLinear, 0},
		{"linear 37", config.GradientLinear, 37},
		{"linear 90", config.GradientLinear, 90},
		{"linear 225", config.GradientLinear, 225},
	} {
		dir := t.TempDir()

		render := func(engine string) string {
			cfg, _ := endToEndConfig(t)
			out := filepath.Join(dir, engine+".mp4")

			cfg.Output = &out
			cfg.Engine = engine
			cfg.GradientType = tc.gradientType
			cfg.LinearAngle = tc.angle
			cfg.Duration = 1
			cfg.Bitrate = 20_000_000
			cfg.Text = new(string) // no overlay: compare the gradients alone

			if err := Encode(context.Background(), cfg, Options{}); err != nil {
				t.Fatalf("%s %s: %v", tc.name, engine, err)
			}

			return out
		}

		native, reference := render(config.EngineNative), render(config.EngineGeq)

		db := psnr(t, native, reference)
		t.Logf("%s: native vs geq PSNR %.1f dB", tc.name, db)

		if db < minPSNR {
			t.Errorf("%s: native vs geq PSNR %.1f dB, want >= %.0f", tc.name, db, minPSNR)
		}
	}
}

// Both engines rasterize the text with the same code and must put it in the
// same place. Inside the text's bounding box the engines agree far better with
// each other than either does with a clip that has no text at all.
func TestOverlayEnginesAgree(t *testing.T) {
	dir := t.TempDir()

	render := func(name, engine, text string) string {
		cfg, _ := endToEndConfig(t)
		out := filepath.Join(dir, name+".mp4")

		cfg.Output = &out
		cfg.Engine = engine
		cfg.Text = &text
		cfg.Duration = 1
		cfg.Bitrate = 20_000_000

		if err := Encode(context.Background(), cfg, Options{}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		return out
	}

	const text = "Hello\nvidgen"

	native := render("native", config.EngineNative, text)
	reference := render("geq", config.EngineGeq, text)
	blank := render("blank", config.EngineNative, "")

	cfg, _ := endToEndConfig(t)
	cfg.Text = new(string)
	*cfg.Text = text

	txt, err := buildText(cfg)
	if err != nil || txt == nil {
		t.Fatalf("buildText: %v, %v", txt, err)
	}

	m := txt.mask
	crop := fmt.Sprintf("crop=%d:%d:%d:%d", m.W, m.H, m.X, m.Y)

	agree := psnrFiltered(t, native, reference, crop)
	vsBlank := psnrFiltered(t, native, blank, crop)
	t.Logf("text box PSNR: native vs geq %.1f dB, native vs no text %.1f dB", agree, vsBlank)

	if agree < vsBlank+8 {
		t.Errorf("engines disagree on text: %.1f dB vs %.1f dB for a clip without text", agree, vsBlank)
	}
}

// An empty text string disables the overlay entirely.
func TestEmptyTextDrawsNothing(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.Text = new(string)

	if txt, err := buildText(cfg); err != nil || txt != nil {
		t.Errorf("buildText = %v, %v; want nil, nil", txt, err)
	}

	cfg.Text = nil // metadata overlay

	txt, err := buildText(cfg)
	if err != nil || txt == nil || txt.mask.Empty() {
		t.Errorf("metadata overlay missing: %v, %v", txt, err)
	}
}

func TestBuildTextRejectsBadColor(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.FontColor = "nothex"

	if _, err := buildText(cfg); err == nil {
		t.Error("bad font_color accepted")
	}
}

func TestEncodeCancelledLeavesNothing(t *testing.T) {
	for _, segments := range []int{1, 3} {
		cfg, out := endToEndConfig(t)
		cfg.Segments = segments

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := Encode(ctx, cfg, Options{})
		if !errors.Is(err, ErrInterrupted) {
			t.Errorf("segments=%d: got %v, want ErrInterrupted", segments, err)
		}

		assertOnlyOutputDirEmpty(t, filepath.Dir(out))
	}
}

func TestEncodeRejectsBadSetup(t *testing.T) {
	cfg, _ := endToEndConfig(t)

	noOutput := cfg
	noOutput.Output = nil

	if Encode(context.Background(), noOutput, Options{}) == nil {
		t.Error("nil output accepted")
	}

	badEngine := cfg
	badEngine.Engine = "bogus"

	if Encode(context.Background(), badEngine, Options{}) == nil {
		t.Error("unknown engine accepted")
	}
}

func assertOnlyOutputDirEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Errorf("want empty dir after cancel, found %d entries (first: %q)", len(entries), entries[0].Name())
	}
}
