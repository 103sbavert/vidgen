package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"sbavert/vidgen/config"
	"sbavert/vidgen/ffmpeg"
	"sbavert/vidgen/progress"

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
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			bar := progress.New(os.Stderr, cfg.TotalFrames())
			stopBar := bar.Start()

			err = ffmpeg.Encode(ctx, cfg, ffmpeg.Options{OnFrame: bar.Tick})
			stopBar()

			if err != nil {
				return err
			}

			fmt.Printf("wrote %s\n", *cfg.Output)

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to JSON config file (defaults when omitted)")

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		if errors.Is(err, ffmpeg.ErrInterrupted) {
			os.Exit(130)
		}

		os.Exit(1)
	}
}
