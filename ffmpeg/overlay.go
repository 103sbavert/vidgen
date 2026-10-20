package ffmpeg

import (
	"fmt"
	"image/png"
	"os"

	"sbavert/vidgen/caption"
	"sbavert/vidgen/config"
)

// text is the rasterized overlay for a clip, shared by both engines so they
// draw identical glyphs.
type text struct {
	mask  *caption.Mask
	color [3]uint8
}

// buildText lays out cfg's overlay text for the clip's frame size. It returns
// nil when there is nothing to draw (empty text or no visible ink).
func buildText(cfg config.VideoConfig) (*text, error) {
	lines := cfg.OverlayLines()
	if len(lines) == 0 {
		return nil, nil
	}

	w, h, err := cfg.Resolution.Size()
	if err != nil {
		return nil, err
	}

	r, g, b, err := cfg.FontColor.ParseHex()
	if err != nil {
		return nil, fmt.Errorf("font_color: %w", err)
	}

	mask, err := caption.Layout(lines, float64(cfg.FontPx()), w, h)
	if err != nil {
		return nil, err
	}

	if mask.Empty() {
		return nil, nil
	}

	return &text{mask: mask, color: [3]uint8{r, g, b}}, nil
}

// writePNG writes the overlay as a full-frame transparent PNG at path, for
// ffmpeg's overlay filter (geq engine).
func (t *text) writePNG(cfg config.VideoConfig, path string) error {
	w, h, err := cfg.Resolution.Size()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := png.Encode(f, t.mask.NRGBA(w, h, t.color)); err != nil {
		f.Close()

		return err
	}

	return f.Close()
}
