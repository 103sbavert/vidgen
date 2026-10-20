package ffmpeg

import (
	"context"
	"fmt"
	"runtime"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

// newSource picks the native frame generator for the configured gradient.
func newSource(cfg config.VideoConfig) (render.Source, error) {
	switch cfg.GradientType {
	case config.GradientRadial:
		return render.NewRadial(cfg)
	default:
		return nil, fmt.Errorf("unsupported gradient type %q", cfg.GradientType)
	}
}

// encodeNative renders frames in Go and pipes them into ffmpeg, as one
// process or as parallel segments depending on planParallelism.
func encodeNative(ctx context.Context, cfg config.VideoConfig, onFrame func(), output string) error {
	src, err := newSource(cfg)
	if err != nil {
		return err
	}

	total := src.Frames()
	p := planParallelism(cfg, total, runtime.NumCPU())

	j := job{
		cfg:     cfg,
		src:     src,
		start:   0,
		end:     total,
		output:  output,
		threads: p.threads,
		workers: p.workers,
		onFrame: onFrame,
	}

	if p.segments > 1 {
		return encodeSegments(ctx, j, p.segments, output)
	}

	return j.run(ctx)
}
