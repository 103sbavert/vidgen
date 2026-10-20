package ffmpeg

import (
	"sbavert/vidgen/config"
)

const (
	// minSegmentFrames keeps auto-chosen segments long enough that the extra
	// keyframe and rate-control restart per segment stay negligible.
	minSegmentFrames = 24

	// maxAutoSegments caps automatic splitting; beyond this the encoders
	// that benefit gain nothing measurable.
	maxAutoSegments = 8
)

// plan is how one clip is spread over the machine.
type plan struct {
	// segments is the number of independent ffmpeg processes (1 = a single
	// process encoding the whole clip).
	segments int
	// workers is the render goroutine count per segment.
	workers int
	// threads is the encoder thread limit per segment; 0 leaves the encoder's
	// default, which is right when a single process owns the machine.
	threads int
}

// planParallelism decides segment, worker and thread counts for a clip of
// totalFrames on a machine with cpus logical CPUs. It is the only place
// these defaults are resolved.
func planParallelism(cfg config.VideoConfig, totalFrames, cpus int) plan {
	segments := cfg.Segments
	if segments == 0 {
		segments = autoSegments(cfg.Codec, totalFrames, cpus)
	}

	segments = max(1, min(segments, totalFrames))

	workers := cfg.Workers
	if workers <= 0 {
		workers = cpus
	}

	p := plan{segments: segments, workers: max(1, workers/segments)}
	if segments > 1 {
		p.threads = max(1, cpus/segments)
	}

	return p
}

// autoSegments picks a segment count when the user left it at 0, from the
// codec's SplitByDefault flag (see config.CodecInfo). Other encoders gain
// nothing from splitting, so they get one segment; libx265 gains ~25% but
// shifts rate control noticeably, so it stays opt-in.
func autoSegments(codec string, totalFrames, cpus int) int {
	info, ok := config.LookupCodec(codec)
	if !ok || !info.SplitByDefault {
		return 1
	}

	return max(1, min(maxAutoSegments, cpus/4, totalFrames/minSegmentFrames))
}
