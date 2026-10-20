package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
// base carries everything shared by the segments (config, source, thread and
// worker limits, callback); its frame range and output are filled in per
// segment. Every segment starts on a keyframe, which is what makes stream-copy
// concat possible.
func encodeSegments(ctx context.Context, base job, n int, output string) error {
	dir, err := os.MkdirTemp(filepath.Dir(output), ".vidgen-segments-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		list     strings.Builder
	)

	ext := filepath.Ext(output)

	for i, r := range splitFrames(base.src.Frames(), n) {
		name := fmt.Sprintf("seg_%03d%s", i, ext)
		fmt.Fprintf(&list, "file '%s'\n", name)

		j := base
		j.start, j.end = r[0], r[1]
		j.output = filepath.Join(dir, name)

		wg.Go(func() {
			err := j.run(ctx)
			if err == nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			if firstErr == nil {
				firstErr = fmt.Errorf("segment %d: %w", i, err)
				cancel()
			}
		})
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	return concat(ctx, dir, list.String(), output)
}

// concat joins the segment files named in list (concat demuxer syntax,
// relative to dir) into output without re-encoding.
func concat(ctx context.Context, dir, list, output string) error {
	listPath := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(listPath, []byte(list), 0o644); err != nil {
		return err
	}

	args := append(quietArgs(), "-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", output)

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("concat segments: %w", err)
	}

	return nil
}
