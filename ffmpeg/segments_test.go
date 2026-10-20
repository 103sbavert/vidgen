package ffmpeg

import (
	"slices"
	"testing"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

func TestSplitFrames(t *testing.T) {
	for _, tc := range []struct{ total, n int }{{720, 4}, {721, 4}, {10, 10}, {7, 3}, {1, 1}} {
		ranges := splitFrames(tc.total, tc.n)

		if len(ranges) != tc.n {
			t.Fatalf("splitFrames(%d,%d): %d ranges", tc.total, tc.n, len(ranges))
		}

		next, smallest, largest := 0, tc.total, 0

		for _, r := range ranges {
			if r[0] != next {
				t.Fatalf("splitFrames(%d,%d): gap or overlap at %v", tc.total, tc.n, r)
			}

			size := r[1] - r[0]
			smallest, largest = min(smallest, size), max(largest, size)
			next = r[1]
		}

		if next != tc.total {
			t.Errorf("splitFrames(%d,%d): covers %d frames", tc.total, tc.n, next)
		}

		if largest-smallest > 1 {
			t.Errorf("splitFrames(%d,%d): unbalanced sizes %d..%d", tc.total, tc.n, smallest, largest)
		}
	}
}

func TestSegmentCount(t *testing.T) {
	cfg := config.GetDefaultConfig()

	if got := segmentCount(cfg, 720); got != 1 {
		t.Errorf("libx264 auto = %d, want 1", got)
	}

	cfg.Segments = 4
	if got := segmentCount(cfg, 720); got != 4 {
		t.Errorf("explicit 4 = %d, want 4", got)
	}

	if got := segmentCount(cfg, 3); got != 3 {
		t.Errorf("explicit 4 on 3 frames = %d, want 3", got)
	}

	cfg.Segments = 0
	cfg.Codec = "libvpx-vp9"
	// Too short to split: never below minSegmentFrames per segment.
	if got := segmentCount(cfg, minSegmentFrames-1); got != 1 {
		t.Errorf("short vp9 clip auto = %d, want 1", got)
	}
}

func TestEncodeArgs(t *testing.T) {
	cfg := config.GetDefaultConfig()
	preset := "veryfast"
	cfg.Preset = &preset

	args := encodeArgs(cfg, render.NewLayout(1920, 1080), "out.mp4", 720, 2)

	for _, want := range [][]string{
		{"-f", "rawvideo"},
		{"-video_size", "1920x1080"},
		{"-framerate", "24"},
		{"-preset", "veryfast"},
		{"-threads", "2"},
		{"-frames:v", "720"},
	} {
		if !containsPair(args, want[0], want[1]) {
			t.Errorf("args missing %v: %v", want, args)
		}
	}

	if args[len(args)-1] != "out.mp4" {
		t.Errorf("output must be last arg, got %q", args[len(args)-1])
	}

	cfg.Preset = nil

	if args := encodeArgs(cfg, render.NewLayout(640, 360), "o.mp4", 1, 0); slices.Contains(args, "-preset") || slices.Contains(args, "-threads") {
		t.Errorf("unset preset/threads must not appear: %v", args)
	}
}

func containsPair(args []string, k, v string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == k && args[i+1] == v {
			return true
		}
	}

	return false
}
