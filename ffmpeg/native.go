package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

// newSource picks the native frame generator for the configured gradient.
func newSource(cfg config.VideoConfig) (render.Source, error) {
	switch cfg.GradientType {
	case "radial":
		return render.NewRadial(cfg)
	default:
		return nil, fmt.Errorf("unsupported gradient type %q", cfg.GradientType)
	}
}

// partialPath returns a hidden sibling of output that keeps its extension (so
// ffmpeg still picks the right muxer): dir/.stem.partial.ext
func partialPath(output string) string {
	dir, file := filepath.Split(output)
	ext := filepath.Ext(file)
	stem := strings.TrimSuffix(file, ext)

	return filepath.Join(dir, "."+stem+".partial"+ext)
}

// renderNative renders frames in Go and pipes them into ffmpeg for encoding.
// The result is written to a temporary file and renamed on success, so a
// failed or interrupted run never leaves a truncated output behind.
func renderNative(cfg config.VideoConfig) error {
	src, err := newSource(cfg)
	if err != nil {
		return err
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	output := *cfg.Output
	tmp := partialPath(output)
	total := src.Frames()

	prog := newProgress(total)
	stopProgress := prog.start()

	err = encodeRange(ctx, cfg, src, 0, total, tmp, 0, prog.Tick)
	stopProgress()

	if err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, output)
}
