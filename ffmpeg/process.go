package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

// job is one ffmpeg process fed by one render pipeline: frames [start, end)
// of src, encoded into output.
type job struct {
	cfg        config.VideoConfig
	src        render.Source
	start, end int
	output     string

	// threads limits the encoder's threads and workers the render
	// goroutines; <= 0 means the defaults.
	threads, workers int

	// onFrame is called after each frame is handed to ffmpeg. May be nil.
	onFrame func()
}

// run starts ffmpeg, streams the frames into its stdin and waits for it.
func (j job) run(ctx context.Context) error {
	args := nativeArgs(j.cfg, j.src.Layout(), j.output, j.end-j.start, j.threads)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	streamErr := render.Stream(ctx, j.src, j.start, j.end, stdin, render.Options{
		Workers:  j.workers,
		Progress: j.onFrame,
	})
	stdin.Close()

	// ffmpeg's own failure is the root cause when it dies early: the writer
	// then only sees a broken pipe.
	if waitErr := cmd.Wait(); waitErr != nil {
		return fmt.Errorf("ffmpeg: %w", waitErr)
	}

	return streamErr
}
