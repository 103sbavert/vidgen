package ffmpeg

import (
	"context"
	"runtime"

	"sbavert/vidgen/config"
	"sbavert/vidgen/gradient"
	"sbavert/vidgen/render"
)

// newSource builds the native frame generator for the configured gradient.
func newSource(cfg config.VideoConfig) (render.Source, error) {
	g, err := gradient.New(cfg)
	if err != nil {
		return nil, err
	}

	w, h, err := cfg.Resolution.Size()
	if err != nil {
		return nil, err
	}

	return render.NewBanded(g, w, h, cfg.TotalFrames())
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
