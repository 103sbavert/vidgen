package configuration

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

var defaultConfig = VideoConfig{
	Resolution:   "1920x1080",
	Framerate:    24,
	Bitrate:      2097152,
	Codec:        "libx264",
	Duration:     30,
	NbColors:     4,
	Speed:        0.08,
	GradientType: "radial",
	LinearAngle:  0,
	FontColor:    "black",
	Colors:       []Color{"546b41", "99ad7a", "dcccac", "fff8ec"},
}

var colorKeywords = map[string]string{
	"black": "222222",
	"white": "eeeeee",
	"red":   "ee2222",
	"green": "22ee22",
	"blue":  "2222ee",
}

var validCodecs = map[string]VideoCodec{
	"libx264":           {Name: "libx264", FormatExtension: ExtMp4},
	"h264_nvenc":        {Name: "h264_nvenc", FormatExtension: ExtMp4},
	"h264_amf":          {Name: "h264_amf", FormatExtension: ExtMp4},
	"h264_qsv":          {Name: "h264_qsv", FormatExtension: ExtMp4},
	"h264_videotoolbox": {Name: "h264_videotoolbox", FormatExtension: ExtMp4},
	"libx265":           {Name: "libx265", FormatExtension: ExtMp4},
	"hevc_nvenc":        {Name: "hevc_nvenc", FormatExtension: ExtMp4},
	"hevc_amf":          {Name: "hevc_amf", FormatExtension: ExtMp4},
	"hevc_qsv":          {Name: "hevc_qsv", FormatExtension: ExtMp4},
	"hevc_videotoolbox": {Name: "hevc_videotoolbox", FormatExtension: ExtMp4},
	"libvpx":            {Name: "libvpx", FormatExtension: ExtWebm},
	"libvpx-vp9":        {Name: "libvpx-vp9", FormatExtension: ExtWebm},
	"vp9_qsv":           {Name: "vp9_qsv", FormatExtension: ExtWebm},
	"libaom-av1":        {Name: "libaom-av1", FormatExtension: ExtMp4},
	"libsvtav1":         {Name: "libsvtav1", FormatExtension: ExtMp4},
	"av1_nvenc":         {Name: "av1_nvenc", FormatExtension: ExtMp4},
	"av1_amf":           {Name: "av1_amf", FormatExtension: ExtMp4},
	"av1_qsv":           {Name: "av1_qsv", FormatExtension: ExtMp4},
	"mpeg2video":        {Name: "mpeg2video", FormatExtension: ExtMpg},
	"mpeg2_qsv":         {Name: "mpeg2_qsv", FormatExtension: ExtMpg},
	"mpeg4":             {Name: "mpeg4", FormatExtension: ExtMp4},
	"libxvid":           {Name: "libxvid", FormatExtension: ExtAvi},
	"prores":            {Name: "prores", FormatExtension: ExtMov},
	"prores_ks":         {Name: "prores_ks", FormatExtension: ExtMov},
	"dnxhd":             {Name: "dnxhd", FormatExtension: ExtMxf},
	"libtheora":         {Name: "libtheora", FormatExtension: ExtOgv},
	"mjpeg":             {Name: "mjpeg", FormatExtension: ExtAvi},
	"mjpeg_qsv":         {Name: "mjpeg_qsv", FormatExtension: ExtAvi},
	"libwebp":           {Name: "libwebp", FormatExtension: ExtWebp},
	"libwebp_anim":      {Name: "libwebp_anim", FormatExtension: ExtWebp},
}

var RegexPatterns = regexPatterns{
	resolutionRegex: regexp.MustCompile(`^(\d{3,4})x(\d{3,4})$`),
	colorRegex:      regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`),
}
