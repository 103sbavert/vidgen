package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func (c *VideoConfig) GetAutomaticFilename() string {
	return fmt.Sprintf("output_%s_%s_%s_%dfps_%bps", c.GradientType, *c.Output, c.Resolution, c.Framerate, c.Bitrate)
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
