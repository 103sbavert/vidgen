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

	return localConfig
}
