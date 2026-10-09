package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func (c *VideoConfig) GetAutomaticFilename() string {
	ext := c.Codec.FormatExtension
	if ext == "" {
		ext = ExtMp4
	}
	return fmt.Sprintf("%s_%s_%dfps_%dbps_%ds%s", c.GradientType, c.Resolution, c.Framerate, c.Bitrate, c.Duration, ext)
}

func (c Color) ParseHex() (r, g, b uint8, err error) {
	if len(c) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q: must be 6 hex digits", string(c))
	}

	rHex := string(c[0:2])
	gHex := string(c[2:4])
	bHex := string(c[4:6])

	rUint, err := strconv.ParseUint(rHex, 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid red hex %s: %w", rHex, err)
	}

	gUint, err := strconv.ParseUint(gHex, 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid green hex %s: %w", gHex, err)
	}

	bUint, err := strconv.ParseUint(bHex, 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid blue hex %s: %w", bHex, err)
	}

	r = uint8(rUint)
	g = uint8(gUint)
	b = uint8(bUint)

	return r, g, b, nil
}

func LoadJSONFile(configPath *string) VideoConfig {
	var localConfig VideoConfig

	bytes, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = json.Unmarshal(bytes, &localConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	localConfig.cyclePalette()

	return localConfig
}

// cyclePalette expands or truncates Colors to NbColors by cycling the palette.
// NbColors wins over len(Colors); a non-positive NbColors leaves Colors as-is.
func (c *VideoConfig) cyclePalette() {
	if c.NbColors < 1 || len(c.Colors) == 0 || len(c.Colors) == c.NbColors {
		return
	}
	cycled := make([]Color, c.NbColors)
	for i := range cycled {
		cycled[i] = c.Colors[i%len(c.Colors)]
	}
	c.Colors = cycled
}
