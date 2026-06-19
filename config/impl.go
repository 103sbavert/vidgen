package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func (c *VideoConfig) GetAutomaticFilename() string {
	return fmt.Sprintf("output_%s_%s_%s_%dfps_%bps", c.GradientType, *c.Output, c.Resolution, c.Framerate, c.Bitrate)
}

func (c Color) ParseHex() (r, g, b uint8, err error) {
	r_hex := string(c[0:2])
	g_hex := string(c[2:4])
	b_hex := string(c[4:6])

	r_uint, err := strconv.ParseUint(r_hex, 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid red hex %s: %w", r_hex, err)
	}

	g_uint, err := strconv.ParseUint(g_hex, 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid green hex %s: %w", g_hex, err)
	}

	b_uint, err := strconv.ParseUint(b_hex, 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid blue hex %s: %w", b_hex, err)
	}

	r = uint8(r_uint)
	g = uint8(g_uint)
	b = uint8(b_uint)

	return r, g, b, nil
}

func LoadJsonFile(configPath *string) VideoConfig {
	local_config := defaultConfig

	file_bytes, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = json.Unmarshal(file_bytes, &local_config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return local_config
}
