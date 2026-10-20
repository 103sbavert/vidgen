package ffmpeg

import (
	"context"
	"os"
	"os/exec"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg/geq"
)

// encodeGeq generates the clip inside a single ffmpeg process with the geq
// filter. Slow; kept as the reference implementation. ffmpeg's own stats go to
// the console, so the progress callback is not used.
func encodeGeq(ctx context.Context, cfg config.VideoConfig, output string) error {
	expr, err := geq.Build(cfg)
	if err != nil {
		return err
	}

	args := geqArgs(cfg, expr, output)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
