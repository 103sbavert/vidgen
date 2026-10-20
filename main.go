package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg"

	"github.com/spf13/cobra"
)

func LoadConfig(p string) (cfg config.VideoConfig, err error) {
	if p == "" {
		cfg = config.GetDefaultConfig()
	} else {
		p, err = filepath.Abs(p)
		if err != nil {
			return cfg, err
		}

		cfg, err = config.ReadConfigFile(&p)
		if err != nil {
			return cfg, err
		}

	}

	if err = cfg.Validate(); err != nil {
		return cfg, err
	}

	err = cfg.CalcOutputPath()
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func main() {
	var configPath string

	rootCmd := &cobra.Command{
		Use:           "vidgen",
		Short:         "Generate synthetic test videos with animated gradient backgrounds",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("ffmpeg"); err != nil {
				return fmt.Errorf("ffmpeg not found on PATH: install ffmpeg to encode video")
			}

			var cfg config.VideoConfig
			var err error

			cfg, err = LoadConfig(configPath)
			if err != nil {
				return err
			}

			if err := ffmpeg.Render(cfg); err != nil {
				return err
			}

			fmt.Printf("wrote %s\n", *cfg.Output)
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to JSON config file (defaults when omitted)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
