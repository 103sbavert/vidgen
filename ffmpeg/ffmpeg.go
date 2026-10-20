// Package ffmpeg turns a video config into an encoded file using the ffmpeg
// binary.
//
// Two engines exist (config.Engine):
//   - "native" (default): frames are rendered in Go on all cores (package
//     render) and piped to ffmpeg as raw yuv420p, so ffmpeg only encodes.
//     Optionally the clip is split into parallel segments (see parallelism.go).
//   - "geq": a single ffmpeg geq filter expression generates the frames
//     (package ffmpeg/geq). Much slower; kept as a reference implementation.
//
// Both write to a temporary file that is renamed on success.
package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sbavert/vidgen/config"
)

// ErrInterrupted is returned by Encode when ctx was cancelled mid-run.
var ErrInterrupted = errors.New("interrupted")

// Options tunes Encode.
type Options struct {
	// OnFrame, if set, is called once per frame handed to the encoder. It may
	// be called from several goroutines at once. The geq engine never calls
	// it (ffmpeg reports its own progress).
	OnFrame func()
}

// Encode renders the video described by cfg to *cfg.Output using cfg.Engine.
// cfg must be validated and have its output path resolved (see config.Load).
// The output only appears once encoding has fully succeeded; a failed or
// cancelled run leaves nothing behind.
func Encode(ctx context.Context, cfg config.VideoConfig, opts Options) error {
	if _, err := exec.LookPath(binary); err != nil {
		return fmt.Errorf("%s not found on PATH: install ffmpeg to encode video", binary)
	}

	if cfg.Output == nil {
		return errors.New("no output path set")
	}

	var encode func(tmp string) error

	switch cfg.Engine {
	case config.EngineNative:
		encode = func(tmp string) error { return encodeNative(ctx, cfg, opts.OnFrame, tmp) }
	case config.EngineGeq:
		encode = func(tmp string) error { return encodeGeq(ctx, cfg, tmp) }
	default:
		return fmt.Errorf("unsupported engine %q", cfg.Engine)
	}

	return writeAtomically(ctx, *cfg.Output, encode)
}

// writeAtomically has encode write to a hidden sibling of output and renames
// it into place on success; on failure the sibling is removed.
func writeAtomically(ctx context.Context, output string, encode func(tmp string) error) error {
	tmp := partialPath(output)

	if err := encode(tmp); err != nil {
		os.Remove(tmp)

		if ctx.Err() != nil {
			return ErrInterrupted
		}

		return err
	}

	return os.Rename(tmp, output)
}

// partialPath returns a hidden sibling of output that keeps its extension (so
// ffmpeg still picks the right muxer): dir/.stem.partial.ext
func partialPath(output string) string {
	dir, file := filepath.Split(output)
	ext := filepath.Ext(file)
	stem := strings.TrimSuffix(file, ext)

	return filepath.Join(dir, "."+stem+".partial"+ext)
}
