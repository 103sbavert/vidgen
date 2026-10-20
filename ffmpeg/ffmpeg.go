// Package ffmpeg turns a video config into an encoded file using the ffmpeg
// binary.
//
// Two engines exist:
//   - "native" (default): frames are rendered in Go on all cores (package
//     render) and piped to ffmpeg as raw yuv420p, so ffmpeg only encodes.
//   - "geq": a single ffmpeg geq filter expression generates the frames
//     (package ffmpeg/geq). Much slower; kept as a reference implementation.
package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg/geq"
)

// Render encodes the video described by cfg to *cfg.Output using cfg.Engine.
func Render(cfg config.VideoConfig) error {
	switch cfg.Engine {
	case "native", "":
		return renderNative(cfg)
	case "geq":
		return renderGeq(cfg)
	default:
		return fmt.Errorf("unsupported engine %q", cfg.Engine)
	}
}

// renderGeq builds the gradient filter expression for cfg and encodes output
// with a single ffmpeg process. Dispatch is by gradient type; extend the
// switch alongside new geq builders.
func renderGeq(cfg config.VideoConfig) error {
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

	return run(buildGeqArgs(cfg, expr))
}

// buildGeqArgs assembles the ffmpeg argv for a gradient clip. The gradient itself
// comes from the geq filter expression; length lives in output -t only (the
// nullsrc input is left endless). No shell is involved, so geq needs no quoting.
func buildGeqArgs(cfg config.VideoConfig, geq string) []string {
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

// run executes argv with output streamed to the console.
func run(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("no command to run")
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
