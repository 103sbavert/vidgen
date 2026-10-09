package main

import (
	"fmt"
	"os"
	"path/filepath"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg/geq"

	"github.com/spf13/cobra"
)

func main() {
	var configPath string

	rootCmd := &cobra.Command{
		Use:           "vidgen",
		Short:         "Generate synthetic test videos with animated gradient backgrounds",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg config.VideoConfig

			if configPath == "" {
				cfg = config.DefaultConfig
			} else {
				absConfPath, err := filepath.Abs(configPath)
				if err != nil {
					return err
				}
				cfg = config.LoadJSONFile(&absConfPath)
			}

			expr, err := geq.BuildRadialGeq(cfg)
			if err != nil {
				return err
			}

			fmt.Printf("%s", expr)
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to JSON config file (defaults when omitted)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
