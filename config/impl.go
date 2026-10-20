package config

import (
	"fmt"
	"os"
	"strconv"
)

// fileExists reports whether path exists. Any stat error counts as absent;
// genuine I/O failures surface when the path is actually used.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

// ParseHex splits a 6-digit hex color into 8-bit RGB channels.
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
