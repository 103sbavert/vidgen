// Package ffmpeg shells out to the ffmpeg binary to encode the final video.
package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg/geq"
)

// Render builds the gradient filter expression for cfg and encodes output.
// Dispatch is by gradient type; extend the switch alongside new geq builders.
func Render(cfg config.VideoConfig) error {
	var expr string

	switch cfg.GradientType {
	case "radial":
		radial, err := geq.BuildRadialGeq(cfg)
		if err != nil {
			return err
		}
		expr = radial
	default:
		return fmt.Errorf("unsupported gradient type %q", cfg.GradientType)
	}

	return Run(BuildArgs(cfg, expr))
}

// BuildArgs assembles the ffmpeg argv for a gradient clip. The gradient itself
// comes from the geq filter expression; length lives in output -t only (the
// nullsrc input is left endless). No shell is involved, so geq needs no quoting.
func BuildArgs(cfg config.VideoConfig, geq string) []string {
	fps := strconv.Itoa(cfg.Framerate)

	return []string{
		"ffmpeg",
		"-f", "lavfi",
		"-i", "nullsrc=s=" + string(cfg.Resolution) + ":r=" + fps,
		"-vf", geq,
		"-c:v", cfg.Codec,
		"-r", fps,
		"-b:v", strconv.Itoa(cfg.Bitrate),
		"-pix_fmt", "yuv420p",
		"-t", strconv.Itoa(cfg.Duration),
		*cfg.Output,
	}
}

// Run executes argv (as built by BuildArgs) with output streamed to the console.
func Run(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("no command to run")
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
