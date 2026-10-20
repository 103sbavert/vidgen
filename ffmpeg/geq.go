package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg/geq"
)

// buildGeq returns the geq filter expression for the configured gradient.
// Extend the switch alongside new geq builders.
func buildGeq(cfg config.VideoConfig) (string, error) {
	switch cfg.GradientType {
	case config.GradientRadial:
		return geq.BuildRadialGeq(cfg)
	default:
		return "", fmt.Errorf("unsupported gradient type %q", cfg.GradientType)
	}
}

// encodeGeq generates the clip inside a single ffmpeg process with the geq
// filter. Slow; kept as the reference implementation. ffmpeg's own stats go to
// the console, so the progress callback is not used.
func encodeGeq(ctx context.Context, cfg config.VideoConfig, output string) error {
	expr, err := buildGeq(cfg)
	if err != nil {
		return err
	}

	args := geqArgs(cfg, expr, output)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
