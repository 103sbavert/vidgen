package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

// encodeArgs builds the ffmpeg argv that reads raw yuv420p frames from stdin
// and encodes frameCount of them into output. threads <= 0 leaves the
// encoder's own default.
//
// Frames are already BT.709 limited-range yuv420p, so no pixel conversion
// happens in ffmpeg; the color tags are only written into the stream.
func encodeArgs(cfg config.VideoConfig, l render.Layout, output string, frameCount, threads int) []string {
	fps := strconv.Itoa(cfg.Framerate)

	args := []string{
		"ffmpeg", "-hide_banner", "-loglevel", "warning", "-nostats", "-y",
		"-f", "rawvideo",
		"-pix_fmt", "yuv420p",
		"-video_size", fmt.Sprintf("%dx%d", l.W, l.H),
		"-framerate", fps,
		"-color_range", "tv",
		"-colorspace", "bt709",
		"-i", "pipe:0",
		"-c:v", cfg.Codec,
		"-b:v", strconv.Itoa(cfg.Bitrate),
	}

	if cfg.Preset != nil && *cfg.Preset != "" {
		args = append(args, "-preset", *cfg.Preset)
	}

	if threads > 0 {
		args = append(args, "-threads", strconv.Itoa(threads))
	}

	return append(args,
		"-pix_fmt", "yuv420p",
		"-color_range", "tv",
		"-colorspace", "bt709",
		"-color_primaries", "bt709",
		"-color_trc", "bt709",
		"-r", fps,
		"-frames:v", strconv.Itoa(frameCount),
		output,
	)
}

// encodeRange renders frames [start, end) of src, pipes them to a fresh
// ffmpeg process and writes the encoded result to output.
func encodeRange(ctx context.Context, cfg config.VideoConfig, src render.Source, start, end int, output string, threads int, onFrame func()) error {
	args := encodeArgs(cfg, src.Layout(), output, end-start, threads)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	streamErr := render.Stream(ctx, src, start, end, stdin, render.Options{
		Workers:  cfg.Workers,
		Progress: onFrame,
	})
	stdin.Close()

	// ffmpeg's own failure is the root cause when it dies early: the writer
	// then only sees a broken pipe.
	if waitErr := cmd.Wait(); waitErr != nil {
		return fmt.Errorf("ffmpeg: %w", waitErr)
	}

	return streamErr
}
