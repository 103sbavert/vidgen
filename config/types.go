package config

import (
	"regexp"
)

type VideoConfig struct {
	Output       *string `json:"output"`
	Resolution   string  `json:"resolution"`
	Framerate    int     `json:"framerate"`
	Bitrate      int     `json:"bitrate"`
	Codec        string  `json:"codec"`
	Duration     int     `json:"duration"`
	Text         *string `json:"text"`
	NbColors     int     `json:"nb_colors"`
	Speed        float64 `json:"speed"`
	GradientType string  `json:"gradient_type"`
	LinearAngle  int     `json:"linear_angle"`
	Seed         *int    `json:"seed"`
	FontSize     *int    `json:"font_size"`
	FontColor    Color   `json:"font_color"`
	Colors       []Color `json:"colors"`
}

type Color string

type VideoResolution string

type VideoCodec struct {
	Name            string
	FormatExtension string
}

type regexPatterns struct {
	resolutionRegex *regexp.Regexp
	colorRegex      *regexp.Regexp
}
