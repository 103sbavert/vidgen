package ffmpeg

import (
	"fmt"

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
