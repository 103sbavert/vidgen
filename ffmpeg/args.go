package ffmpeg

import (
	"fmt"
	"strconv"

	"sbavert/vidgen/config"
	"sbavert/vidgen/render"
)

const binary = "ffmpeg"

// BT.709 limited-range tags. The native renderer produces exactly this, so
// the tags describe the data rather than request a conversion.
var (
	bt709InputTags  = []string{"-color_range", "tv", "-colorspace", "bt709"}
	bt709OutputTags = []string{
		"-color_range", "tv",
		"-colorspace", "bt709",
		"-color_primaries", "bt709",
		"-color_trc", "bt709",
	}
)

// quietArgs makes ffmpeg overwrite its (temporary) output and only print
// warnings; progress is reported by our own pipeline.
func quietArgs() []string {
	return []string{"-hide_banner", "-loglevel", "warning", "-nostats", "-y"}
}

// codecArgs are the encoder options shared by every engine: codec, bitrate,
// optional preset, optional thread limit, pixel format and frame rate.
// threads <= 0 leaves the encoder's own default.
func codecArgs(cfg config.VideoConfig, threads int) []string {
	args := []string{
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
		"-r", strconv.Itoa(cfg.Framerate),
	)
}

// nativeArgs builds the argv that reads raw yuv420p frames from stdin and
// encodes frameCount of them into output.
func nativeArgs(cfg config.VideoConfig, l render.Layout, output string, frameCount, threads int) []string {
	args := append([]string{binary}, quietArgs()...)

	args = append(args,
		"-f", "rawvideo",
		"-pix_fmt", "yuv420p",
		"-video_size", fmt.Sprintf("%dx%d", l.W, l.H),
		"-framerate", strconv.Itoa(cfg.Framerate),
	)
	args = append(args, bt709InputTags...)
	args = append(args, "-i", "pipe:0")
	args = append(args, codecArgs(cfg, threads)...)
	args = append(args, bt709OutputTags...)

	return append(args, "-frames:v", strconv.Itoa(frameCount), output)
}

// geqArgs builds the argv for the geq engine: a lavfi null source filtered by
// the gradient expression. Length lives in output -t only (the nullsrc input
// is left endless). No shell is involved, so geq needs no quoting. Colors go
// through ffmpeg's default RGB->YUV conversion, so no color tags are written.
//
// A non-empty overlayPNG (full-frame, transparent) is composited over the
// gradient with ffmpeg's overlay filter; ffmpeg repeats the single image for
// the whole clip.
func geqArgs(cfg config.VideoConfig, expr, overlayPNG, output string) []string {
	args := []string{
		binary, "-y",
		"-f", "lavfi",
		"-i", "nullsrc=s=" + string(cfg.Resolution) + ":r=" + strconv.Itoa(cfg.Framerate),
	}

	if overlayPNG == "" {
		args = append(args, "-vf", expr)
	} else {
		args = append(args,
			"-i", overlayPNG,
			// The explicit format keeps overlay from negotiating a different
			// layout for the gradient, which thickens the text.
			"-filter_complex", "[0:v]"+expr+",format=gbrp[bg];[bg][1:v]overlay=format=auto[out]",
			"-map", "[out]",
		)
	}

	args = append(args, codecArgs(cfg, 0)...)

	return append(args, "-t", strconv.Itoa(cfg.Duration), output)
}
