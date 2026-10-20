package config

import "regexp"

const (
	ExtMp4  = ".mp4"
	ExtWebm = ".webm"
	ExtMpg  = ".mpg"
	ExtAvi  = ".avi"
	ExtMov  = ".mov"
	ExtMxf  = ".mxf"
	ExtOgv  = ".ogv"
	ExtWebp = ".webp"
)

var colorKeywords = map[string]string{
	"black": "222222",
	"white": "eeeeee",
	"red":   "ee2222",
	"green": "22ee22",
	"blue":  "2222ee",
}

var codecExtMap = map[string]string{
	"libx264":           ExtMp4,
	"h264_nvenc":        ExtMp4,
	"h264_amf":          ExtMp4,
	"h264_qsv":          ExtMp4,
	"h264_videotoolbox": ExtMp4,
	"libx265":           ExtMp4,
	"hevc_nvenc":        ExtMp4,
	"hevc_amf":          ExtMp4,
	"hevc_qsv":          ExtMp4,
	"hevc_videotoolbox": ExtMp4,
	"libvpx":            ExtWebm,
	"libvpx-vp9":        ExtWebm,
	"vp9_qsv":           ExtWebm,
	"libaom-av1":        ExtMp4,
	"libsvtav1":         ExtMp4,
	"av1_nvenc":         ExtMp4,
	"av1_amf":           ExtMp4,
	"av1_qsv":           ExtMp4,
	"mpeg2video":        ExtMpg,
	"mpeg2_qsv":         ExtMpg,
	"mpeg4":             ExtMp4,
	"libxvid":           ExtAvi,
	"prores":            ExtMov,
	"prores_ks":         ExtMov,
	"dnxhd":             ExtMxf,
	"libtheora":         ExtOgv,
	"mjpeg":             ExtAvi,
	"mjpeg_qsv":         ExtAvi,
	"libwebp":           ExtWebp,
	"libwebp_anim":      ExtWebp,
}

var RegexPatterns = regexPatterns{
	resolutionRegex: regexp.MustCompile(`^(\d{3,4})x(\d{3,4})$`),
	colorRegex:      regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`),
}
