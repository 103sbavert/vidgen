package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OverlayLines returns the text drawn over the video, one entry per row:
//   - Text unset (null): clip metadata — resolution, frame rate, container
//     format, codec and duration.
//   - Text set: its lines (split on "\n").
//   - Text empty: nil, meaning no overlay at all.
func (c VideoConfig) OverlayLines() []string {
	if c.Text != nil {
		if *c.Text == "" {
			return nil
		}

		return strings.Split(*c.Text, "\n")
	}

	w, h, err := c.Resolution.Size()
	resolution := string(c.Resolution)
	if err == nil {
		resolution = fmt.Sprintf("%d x %d", w, h)
	}

	return []string{
		resolution + " px",
		fmt.Sprintf("%d fps", c.Framerate),
		c.containerFormat(),
		c.Codec,
		fmt.Sprintf("%ds", c.Duration),
	}
}

// containerFormat is the upper-cased container extension of the output file
// (e.g. "MP4"), or the codec's default container before the output is known.
func (c VideoConfig) containerFormat() string {
	ext := ExtMp4
	if c.Output != nil {
		ext = filepath.Ext(*c.Output)
	} else if info, ok := LookupCodec(c.Codec); ok {
		ext = info.Ext
	}

	if ext == "" {
		return "UNKNOWN"
	}

	return strings.ToUpper(strings.TrimPrefix(ext, "."))
}

// FontPx is the overlay font size in pixels: FontSize when set, otherwise
// scaled to the frame height — height/12 (at least 24) for custom text and the
// smaller height/28 (at least 14) for the multi-line metadata block.
func (c VideoConfig) FontPx() int {
	if c.FontSize != nil {
		return *c.FontSize
	}

	_, h, err := c.Resolution.Size()
	if err != nil {
		return 24
	}

	if c.Text != nil {
		return max(24, h/12)
	}

	return max(14, h/28)
}
