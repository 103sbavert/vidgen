package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

// splitFrames divides [0,total) into n contiguous ranges whose sizes differ
// by at most one frame.
func splitFrames(total, n int) [][2]int {
	ranges := make([][2]int, 0, n)
	base, extra := total/n, total%n
	start := 0

	for i := range n {
		size := base
		if i < extra {
			size++
		}

		ranges = append(ranges, [2]int{start, start + size})
		start += size
	}

	return ranges
}

// encodeSegments encodes the clip as n independent segments in parallel and
// joins them losslessly (concat demuxer, stream copy) into output.
//
// Every segment is a separate ffmpeg process fed by its own render pipeline,
// so encoders that scale poorly across threads can still use every core.
// Each segment starts on a keyframe, which is what makes stream-copy concat
// possible.
func encodeSegments(ctx context.Context, cfg config.VideoConfig, src render.Source, n int, output string, prog *progress) error {
	dir, err := os.MkdirTemp(filepath.Dir(output), ".vidgen-segments-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cpus := runtime.NumCPU()
	threads := max(1, cpus/n)

	workers := cfg.Workers
	if workers <= 0 {
		workers = cpus
	}

	workers = max(1, workers/n)

	ranges := splitFrames(src.Frames(), n)
	ext := filepath.Ext(output)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		list     strings.Builder
	)

	for i, r := range ranges {
		name := fmt.Sprintf("seg_%03d%s", i, ext)
		fmt.Fprintf(&list, "file '%s'\n", name)

		wg.Go(func() {
			err := encodeRange(ctx, cfg, src, r[0], r[1], filepath.Join(dir, name), threads, workers, prog.Tick)
			if err == nil {
				return
			}

			mu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("segment %d: %w", i, err)
				cancel()
			}
			mu.Unlock()
		})
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	listPath := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(listPath, []byte(list.String()), 0o644); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "warning", "-nostats", "-y",
		"-f", "concat", "-safe", "0", "-i", listPath,
		"-c", "copy",
		output,
	)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("concat segments: %w", err)
	}

	return nil
}
