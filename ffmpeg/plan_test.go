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

func TestPlanParallelism(t *testing.T) {
	cfg := func(codec string, segments, workers int) config.VideoConfig {
		c := config.GetDefaultConfig()
		c.Codec, c.Segments, c.Workers = codec, segments, workers

		return c
	}

	for _, tc := range []struct {
		name   string
		cfg    config.VideoConfig
		frames int
		cpus   int
		want   plan
	}{
		{"x264 stays one process", cfg("libx264", 0, 0), 720, 16, plan{segments: 1, workers: 16, threads: 0}},
		{"vp9 auto splits", cfg("libvpx-vp9", 0, 0), 720, 16, plan{segments: 4, workers: 4, threads: 4}},
		{"vp9 capped at 8", cfg("libvpx-vp9", 0, 0), 720, 64, plan{segments: 8, workers: 8, threads: 8}},
		{"vp9 on small machine", cfg("libvpx-vp9", 0, 0), 720, 4, plan{segments: 1, workers: 4, threads: 0}},
		{"vp9 short clip", cfg("libvpx-vp9", 0, 0), minSegmentFrames - 1, 16, plan{segments: 1, workers: 16, threads: 0}},
		{"explicit overrides auto", cfg("libx264", 4, 0), 720, 16, plan{segments: 4, workers: 4, threads: 4}},
		{"explicit clamps to frames", cfg("libx264", 4, 0), 3, 16, plan{segments: 3, workers: 5, threads: 5}},
		{"explicit one disables auto", cfg("libvpx-vp9", 1, 0), 720, 16, plan{segments: 1, workers: 16, threads: 0}},
		{"workers respected", cfg("libx264", 0, 2), 720, 16, plan{segments: 1, workers: 2, threads: 0}},
		{"workers split per segment", cfg("libx264", 2, 6), 720, 16, plan{segments: 2, workers: 3, threads: 8}},
		{"never below one worker", cfg("libx264", 8, 2), 720, 16, plan{segments: 8, workers: 1, threads: 2}},
	} {
		if got := planParallelism(tc.cfg, tc.frames, tc.cpus); got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestNativeArgs(t *testing.T) {
	cfg := config.GetDefaultConfig()
	preset := "veryfast"
	cfg.Preset = &preset

	args := nativeArgs(cfg, render.NewLayout(1920, 1080), "out.mp4", 720, 2)

	for _, want := range [][]string{
		{"-f", "rawvideo"},
		{"-video_size", "1920x1080"},
		{"-framerate", "24"},
		{"-c:v", "libx264"},
		{"-preset", "veryfast"},
		{"-threads", "2"},
		{"-colorspace", "bt709"},
		{"-color_trc", "bt709"},
		{"-frames:v", "720"},
	} {
		if !containsPair(args, want[0], want[1]) {
			t.Errorf("args missing %v: %v", want, args)
		}
	}

	if args[0] != binary || args[len(args)-1] != "out.mp4" {
		t.Errorf("want ffmpeg ... out.mp4, got %v", args)
	}

	cfg.Preset = nil

	args = nativeArgs(cfg, render.NewLayout(640, 360), "o.mp4", 1, 0)
	if slices.Contains(args, "-preset") || slices.Contains(args, "-threads") {
		t.Errorf("unset preset/threads must not appear: %v", args)
	}
}

func TestGeqArgsSharePresetAndSkipColorTags(t *testing.T) {
	cfg := config.GetDefaultConfig()
	preset := "fast"
	cfg.Preset = &preset

	args := geqArgs(cfg, "geq=r=1", "o.mp4")

	for _, want := range [][]string{
		{"-vf", "geq=r=1"},
		{"-c:v", "libx264"},
		{"-preset", "fast"},
		{"-t", "30"},
	} {
		if !containsPair(args, want[0], want[1]) {
			t.Errorf("args missing %v: %v", want, args)
		}
	}

	if slices.Contains(args, "-colorspace") {
		t.Errorf("geq output must stay untagged: %v", args)
	}
}

func TestPartialPath(t *testing.T) {
	for in, want := range map[string]string{
		"out.mp4":         ".out.partial.mp4",
		"dir/clip.webm":   "dir/.clip.partial.webm",
		"/a/b/x.y.mov":    "/a/b/.x.y.partial.mov",
		"/abs/noext":      "/abs/.noext.partial",
		"rel/dir/f_1.mp4": "rel/dir/.f_1.partial.mp4",
	} {
		if got := partialPath(in); got != want {
			t.Errorf("partialPath(%q) = %q, want %q", in, got, want)
		}
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
