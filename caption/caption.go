// Package caption lays out static text and rasterizes it into an anti-aliased
// coverage mask. It knows nothing about video formats: package render blends
// the mask into yuv420p frames, and the geq engine hands ffmpeg the same mask
// as a PNG, so both engines draw identical text.
package caption

import (
	"fmt"
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"sbavert/vidgen/fonts"
)

// Mask is the text coverage of a frame: Alpha holds W*H coverage values
// (0 = untouched, 255 = fully text) for the rectangle whose top-left corner
// is (X, Y) in frame coordinates. The rectangle is the tight bounding box of
// the ink, always inside the frame.
type Mask struct {
	X, Y, W, H int
	Alpha      []uint8
}

// Empty reports whether the mask draws nothing.
func (m *Mask) Empty() bool {
	return m == nil || m.W == 0 || m.H == 0
}

// NRGBA renders the mask as a full-frame, fully transparent image with the
// text in color rgb, ready to be encoded as PNG and overlaid by ffmpeg.
func (m *Mask) NRGBA(frameW, frameH int, rgb [3]uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, frameW, frameH))

	if m.Empty() {
		return img
	}

	for row := range m.H {
		for col := range m.W {
			a := m.Alpha[row*m.W+col]
			if a == 0 {
				continue
			}

			o := img.PixOffset(m.X+col, m.Y+row)
			img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = rgb[0], rgb[1], rgb[2], a
		}
	}

	return img
}

var (
	parsedOnce sync.Once
	parsedFont *opentype.Font
	parsedErr  error
)

func loadFont() (*opentype.Font, error) {
	parsedOnce.Do(func() {
		parsedFont, parsedErr = opentype.Parse(fonts.NotoSansRegular)
	})

	return parsedFont, parsedErr
}

// Layout centers lines (one per row, top to bottom) in a frameW x frameH
// frame at the given font size in pixels and rasterizes them. Each line is
// centered horizontally and the block of lines vertically. Text larger than the
// frame is clipped. The mask is empty when there is no visible ink.
func Layout(lines []string, sizePx float64, frameW, frameH int) (*Mask, error) {
	if sizePx <= 0 {
		return nil, fmt.Errorf("font size must be positive, got %v", sizePx)
	}

	if frameW <= 0 || frameH <= 0 {
		return nil, fmt.Errorf("invalid frame size %dx%d", frameW, frameH)
	}

	if len(lines) == 0 {
		return &Mask{}, nil
	}

	f, err := loadFont()
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}

	// DPI 72 makes the point size equal the pixel size of the em square.
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: sizePx, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, fmt.Errorf("create font face: %w", err)
	}
	defer face.Close()

	metrics := face.Metrics()
	ascent, descent := metrics.Ascent.Round(), metrics.Descent.Round()
	lineHeight := metrics.Height.Round()

	widths := make([]int, len(lines))
	widest := 0

	for i, line := range lines {
		widths[i] = font.MeasureString(face, line).Round()
		widest = max(widest, widths[i])
	}

	// Block rectangle in frame coordinates, clipped to the frame.
	blockTop := (frameH - lineHeight*len(lines)) / 2
	block := image.Rect(
		(frameW-widest)/2, blockTop,
		(frameW-widest)/2+widest, blockTop+lineHeight*len(lines),
	).Intersect(image.Rect(0, 0, frameW, frameH))

	if block.Empty() {
		return &Mask{}, nil
	}

	canvas := image.NewAlpha(image.Rect(0, 0, block.Dx(), block.Dy()))

	for i, line := range lines {
		baseline := blockTop + i*lineHeight + (lineHeight-(ascent+descent))/2 + ascent
		x := (frameW - widths[i]) / 2

		drawer := font.Drawer{
			Dst:  canvas,
			Src:  image.Opaque,
			Face: face,
			Dot:  fixed.P(x-block.Min.X, baseline-block.Min.Y),
		}
		drawer.DrawString(line)
	}

	return trim(canvas, block.Min), nil
}

// trim crops canvas (whose top-left sits at origin in the frame) to the
// bounding box of its non-zero coverage.
func trim(canvas *image.Alpha, origin image.Point) *Mask {
	b := canvas.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if canvas.AlphaAt(x, y).A == 0 {
				continue
			}

			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}
	}

	if maxX < minX {
		return &Mask{}
	}

	m := &Mask{
		X: origin.X + minX, Y: origin.Y + minY,
		W: maxX - minX + 1, H: maxY - minY + 1,
	}
	m.Alpha = make([]uint8, m.W*m.H)

	for row := range m.H {
		src := canvas.Pix[(minY+row)*canvas.Stride+minX:]
		copy(m.Alpha[row*m.W:(row+1)*m.W], src[:m.W])
	}

	return m
}
