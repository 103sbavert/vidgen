package gradient

import "math"

// Shape maps pixel (x, y) of a frame to a normalized position s in 0→1: 0 is
// where the last (innermost) color sits, 1 where the first (outermost) color
// sits. Pixel coordinates are integers, exactly as ffmpeg's geq sees X and Y.
// Values may stray marginally outside 0..1 from rounding; consumers clamp.
type Shape func(x, y int) float64

// RadialShape is the distance from the frame center, normalized so the corner
// is 1. Rings grow outward from the center.
func RadialShape(w, h int) Shape {
	cx, cy := float64(w)/2, float64(h)/2
	norm := 1 / math.Hypot(cx, cy)

	return func(x, y int) float64 {
		return math.Hypot(float64(x)-cx, float64(y)-cy) * norm
	}
}

// Direction returns the unit vector of a gradient angle in degrees measured in
// pixel space: 0 points right (+x), 90 points down (+y). Components are
// rounded to 6 decimals so the native renderer and the geq expression, which
// prints them, use bit-identical constants.
func Direction(angleDeg int) (ux, uy float64) {
	rad := float64(angleDeg) * math.Pi / 180

	return roundTo6(math.Cos(rad)), roundTo6(math.Sin(rad))
}

// LinearShape is a stripe gradient running along angleDeg (see Direction). The
// first color sits at the start of the frame and the last at the far end; the
// position is normalized by the frame's extent along the direction, so the
// full palette always spans corner to corner at any angle.
func LinearShape(w, h, angleDeg int) Shape {
	ux, uy := Direction(angleDeg)
	cx, cy := float64(w)/2, float64(h)/2
	extent := math.Abs(float64(w)*ux) + math.Abs(float64(h)*uy)

	return func(x, y int) float64 {
		along := 0.5 + ((float64(x)-cx)*ux+(float64(y)-cy)*uy)/extent

		return 1 - along
	}
}

func roundTo6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}
