package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg/geq"
)

// encodeGeq generates the clip inside a single ffmpeg process with the geq
// filter. Slow; kept as the reference implementation. ffmpeg's own stats go to
// the console, so the progress callback is not used.
//
// Overlay text is rasterized by the same code as the native engine and handed
// to ffmpeg as a transparent PNG for its overlay filter.
func encodeGeq(ctx context.Context, cfg config.VideoConfig, output string) error {
	expr, err := geq.Build(cfg)
	if err != nil {
		return err
	}

	txt, err := buildText(cfg)
	if err != nil {
		return err
	}

	var overlayPNG string

	if txt != nil {
		dir, err := os.MkdirTemp(filepath.Dir(output), ".vidgen-text-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)

		overlayPNG = filepath.Join(dir, "text.png")
		if err := txt.writePNG(cfg, overlayPNG); err != nil {
			return err
		}
	}

	args := geqArgs(cfg, expr, overlayPNG, output)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
