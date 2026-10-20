package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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

// renderNative renders frames in Go and pipes them into ffmpeg for encoding,
// optionally as several parallel segments (see segmentCount).
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
	segments := segmentCount(cfg, total)

	prog := newProgress(total)
	stopProgress := prog.start()

	if segments > 1 {
		err = encodeSegments(ctx, cfg, src, segments, tmp, prog)
	} else {
		err = encodeRange(ctx, cfg, src, 0, total, tmp, 0, cfg.Workers, prog.Tick)
	}

	stopProgress()

	if err != nil {
		os.Remove(tmp)

		if ctx.Err() != nil {
			return fmt.Errorf("interrupted")
		}

		return err
	}

	return os.Rename(tmp, output)
}

// minSegmentFrames keeps auto-chosen segments long enough that the extra
// keyframe and rate-control restart per segment stay negligible.
const minSegmentFrames = 24

// segmentCount resolves cfg.Segments to the number of parallel segments to
// use for a clip of total frames: 1 means a single ffmpeg process.
func segmentCount(cfg config.VideoConfig, total int) int {
	if cfg.Segments > 0 {
		return max(1, min(cfg.Segments, total))
	}

	return max(1, min(autoSegments(cfg), total/minSegmentFrames))
}

// autoSegments picks a segment count when the user left it at 0.
//
// Measured on 1080p: libx264, libsvtav1, mpeg4 and hardware encoders already
// saturate the machine from one process, so segmenting only costs bitrate
// fidelity. libvpx (VP8/VP9) and libaom scale poorly across threads and gain
// 3-5x from running independent segments side by side. libx265 gains ~25% but
// shifts rate control noticeably, so it stays opt-in (segments: 2).
func autoSegments(cfg config.VideoConfig) int {
	switch cfg.Codec {
	case "libvpx", "libvpx-vp9", "libaom-av1":
		return min(8, max(1, runtime.NumCPU()/4))
	default:
		return 1
	}
}
