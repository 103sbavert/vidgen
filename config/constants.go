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

// Frame generators selectable through VideoConfig.Engine. Keep in sync with
// the `oneof` validation tag on that field.
const (
	EngineNative = "native"
	EngineGeq    = "geq"
)

// Gradient types selectable through VideoConfig.GradientType. Keep in sync
// with the `oneof` validation tag on that field.
const (
	GradientRadial = "radial"
)

var colorKeywords = map[string]string{
	"black": "222222",
	"white": "eeeeee",
	"red":   "ee2222",
	"green": "22ee22",
	"blue":  "2222ee",
}

// CodecInfo is what vidgen knows about an ffmpeg video encoder.
type CodecInfo struct {
	Ext            string
	SplitByDefault bool
}

// codecs lists every encoder accepted by config validation.
var codecs = map[string]CodecInfo{
	"libx264":           {Ext: ExtMp4},
	"h264_nvenc":        {Ext: ExtMp4},
	"h264_amf":          {Ext: ExtMp4},
	"h264_qsv":          {Ext: ExtMp4},
	"h264_videotoolbox": {Ext: ExtMp4},
	"libx265":           {Ext: ExtMp4},
	"hevc_nvenc":        {Ext: ExtMp4},
	"hevc_amf":          {Ext: ExtMp4},
	"hevc_qsv":          {Ext: ExtMp4},
	"hevc_videotoolbox": {Ext: ExtMp4},
	"libvpx":            {Ext: ExtWebm, SplitByDefault: true},
	"libvpx-vp9":        {Ext: ExtWebm, SplitByDefault: true},
	"vp9_qsv":           {Ext: ExtWebm},
	"libaom-av1":        {Ext: ExtMp4, SplitByDefault: true},
	"libsvtav1":         {Ext: ExtMp4},
	"av1_nvenc":         {Ext: ExtMp4},
	"av1_amf":           {Ext: ExtMp4},
	"av1_qsv":           {Ext: ExtMp4},
	"mpeg2video":        {Ext: ExtMpg},
	"mpeg2_qsv":         {Ext: ExtMpg},
	"mpeg4":             {Ext: ExtMp4},
	"libxvid":           {Ext: ExtAvi},
	"prores":            {Ext: ExtMov},
	"prores_ks":         {Ext: ExtMov},
	"dnxhd":             {Ext: ExtMxf},
	"libtheora":         {Ext: ExtOgv},
	"mjpeg":             {Ext: ExtAvi},
	"mjpeg_qsv":         {Ext: ExtAvi},
	"libwebp":           {Ext: ExtWebp},
	"libwebp_anim":      {Ext: ExtWebp},
}

// LookupCodec returns what is known about the named encoder.
func LookupCodec(name string) (CodecInfo, bool) {
	info, ok := codecs[name]

	return info, ok
}

var RegexPatterns = regexPatterns{
	resolutionRegex: regexp.MustCompile(`^(\d{3,4})x(\d{3,4})$`),
	colorRegex:      regexp.MustCompile(`^([0-9a-fA-F]{6})$`),
}
