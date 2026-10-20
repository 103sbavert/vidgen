// Package geq builds ffmpeg geq filter expressions for animated gradients.
//
// The geq filter evaluates a per-pixel expression to determine RGB values.
// This package generates those expressions for banded gradients (radial rings
// or linear stripes) whose bands start at different widths and animate toward
// equal size over the duration of the video. It is the slow reference engine:
// package render is the fast one, and both are driven by package gradient.
//
// Key concepts:
//   - Position (s): normalized 0→1 per pixel, from the shape (radial or linear)
//   - Progress (p): normalized 0→1 from clip start to end
//   - Boundaries: dividing lines between bands, animated via linear interpolation
//   - Smoothstep: S-curve blending to avoid harsh transitions
//   - Color deltas: RGB shifts applied when crossing each boundary
//
// The output is a string like:
//
//	geq=r='st(0,...)...255-35*st(5,...)...':g='...':b='...'
//
// suitable for ffmpeg's -vf flag.
package geq
