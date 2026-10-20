package gradient

import (
	"fmt"

	"sbavert/vidgen/config"
)

// Gradient is a complete animated gradient for one clip: where each pixel sits
// (Shape) and what color that position has at a given time (Bands).
type Gradient struct {
	Bands Bands
	Shape Shape
}

// New builds the gradient selected by cfg.GradientType, sized for
// cfg.Resolution. Adding a gradient type means adding a Shape and a case here
// (and a matching expression in ffmpeg/geq for the reference engine).
func New(cfg config.VideoConfig) (Gradient, error) {
	bands, err := NewBands(cfg.Colors)
	if err != nil {
		return Gradient{}, err
	}

	w, h, err := cfg.Resolution.Size()
	if err != nil {
		return Gradient{}, err
	}

	switch cfg.GradientType {
	case config.GradientRadial:
		return Gradient{Bands: bands, Shape: RadialShape(w, h)}, nil
	case config.GradientLinear:
		return Gradient{Bands: bands, Shape: LinearShape(w, h, cfg.LinearAngle)}, nil
	default:
		return Gradient{}, fmt.Errorf("unsupported gradient type %q", cfg.GradientType)
	}
}
