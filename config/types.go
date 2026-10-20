// Package config defines, loads, and validates the video specification
// used by vidgen.
package config

import (
	"regexp"
)

type VideoConfig struct {
	Output       *string         `json:"output"`
	Resolution   VideoResolution `json:"resolution" validate:"required,resolution"`
	Framerate    int             `json:"framerate" validate:"gt=0,lte=120"`
	Bitrate      int             `json:"bitrate" validate:"gt=0"`
	Codec        string          `json:"codec" validate:"required,codec"`
	Duration     int             `json:"duration" validate:"gt=0"`
	Text         *string         `json:"text"`
	NbColors     int             `json:"nb_colors" validate:"gt=0,lte=16"`
	Speed        float64         `json:"speed" validate:"gt=0"`
	GradientType string          `json:"gradient_type" validate:"required,oneof=radial linear"`
	LinearAngle  int             `json:"linear_angle" validate:"min=0,max=360"`
	Seed         *int            `json:"seed"`
	FontSize     *int            `json:"font_size" validate:"omitempty,gt=0"`
	FontColor    Color           `json:"font_color" validate:"required,color"`
	Colors       []Color         `json:"colors" validate:"dive,color"`
	Engine       string          `json:"engine" validate:"required,oneof=native geq"`
	Workers      int             `json:"workers" validate:"gte=0"`
	Segments     int             `json:"segments" validate:"gte=0,lte=64"`
	Preset       *string         `json:"preset"`
}

type Color string

type VideoResolution string

type regexPatterns struct {
	resolutionRegex *regexp.Regexp
	colorRegex      *regexp.Regexp
}
