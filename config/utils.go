package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// getAutomaticFilename derives the deterministic output name
// ({gradient}_{res}_{fps}fps_{bitrate}bps_{duration}s{ext}) with the extension
// from the codec map, falling back to .mp4.
func getAutomaticFilename(c VideoConfig) string {
	ext := codecExtMap[c.Codec]
	if ext == "" {
		ext = ExtMp4
	}
	return fmt.Sprintf("%s_%s_%dfps_%dbps_%ds%s", c.GradientType, c.Resolution, c.Framerate, c.Bitrate, c.Duration, ext)
}

// GetDefaultConfig builds the base configuration. Missing or null JSON keys
// keep these values; running with no config file uses it as-is.
func GetDefaultConfig() VideoConfig {
	return VideoConfig{
		Resolution:   VideoResolution("1920x1080"),
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
		Engine:       "native",
	}
}

// ReadConfigFile reads the JSON config at p (path-like string) over a copy of
// GetDefaultConfig, so missing keys keep defaults, then normalizes the palette to
// NbColors. Unreadable or invalid input returns an error.
func ReadConfigFile(p *string) (cfg VideoConfig, err error) {
	cfg = GetDefaultConfig()

	bytes, err := os.ReadFile(*p)
	if err != nil {
		err = fmt.Errorf("failed to open config file: %w", err)
		return
	}

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal config JSON: %w", err)
		return
	}

	cfg.cyclePalette()
	return
}

// nextAvailableFilename returns the first free automatic file name: the
// getAutomaticFilename() result, with a numbered suffix (_1, _2, ...) appended
// to the stem if that name is taken.
func nextAvailableFilename(c VideoConfig) string {
	base := getAutomaticFilename(c)

	if !fileExists(base) {
		return base
	}

	stem, ext := base, ""
	if i := strings.LastIndex(base, "."); i >= 0 {
		stem, ext = base[:i], base[i:]
	}

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, i, ext)
		if !fileExists(candidate) {
			return candidate
		}
	}
}
