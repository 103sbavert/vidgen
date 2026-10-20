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

// hexStrToUint converts a `hex` given string into a int64 go integer
func hexStrToUint(hex string) (uint8, error) {
	d, err := strconv.ParseUint(hex, 16, 8)
	if err != nil {
		return 0, err
	}

	return uint8(d), nil
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
		err = fmt.Errorf("invalid hex color %q: must be 6 hex digits", string(c))
		return
	}

	rHex := string(c[0:2])
	gHex := string(c[2:4])
	bHex := string(c[4:6])

	r, err = hexStrToUint(rHex)
	if err != nil {
		err = fmt.Errorf("invalid red hex %s: %w", rHex, err)
		return
	}

	g, err = hexStrToUint(gHex)
	if err != nil {
		err = fmt.Errorf("invalid green hex %s: %w", gHex, err)
		return
	}

	b, err = hexStrToUint(bHex)
	if err != nil {
		err = fmt.Errorf("invalid blue hex %s: %w", bHex, err)
		return
	}

	return
}

// CalcOutputPath intelligently calculates and sets `VideoConfig.Output`. If a
// value is set and a file at the resolved path exists, an error is thrown. An
// automatic file name is determined.
func (c *VideoConfig) CalcOutputPath() error {
	if c.Output == nil {
		p := nextAvailableFilename(*c)
		c.Output = &p
	}

	if _, err := os.Stat(*c.Output); err == nil {
		return fmt.Errorf("output file already exists: %s", *c.Output)
	} else if !os.IsNotExist(err) {
		return err
	}

	return nil
}
