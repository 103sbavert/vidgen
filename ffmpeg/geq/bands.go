package geq

import (
	"fmt"
	"strings"

	"sbavert/vidgen/config"
	"sbavert/vidgen/gradient"
)

// buildSmoothstepTerm builds one geq string fragment representing a single fence's color contribution
// to one channel. This fragment is one term in the final addition/subtraction chain.
//
// It does two things in one expression:
//  1. Compute how far past this fence the pixel is (0=inside, 1=outside, S-curve in between)
//  2. Multiply that 0→1 value by the delta to get the actual channel shift
//
// The blur zone around the fence spans (fence - sigma) to (fence + sigma).
// sigma is the half-width of the fuzzy transition region.
//
// boundarySlot: which ld() slot holds this fence's current position
//
//	(slot 2 for fence 1, slot 3 for fence 2, etc.)
//
// tempSlot:     a scratch slot to temporarily hold the smoothstep t value.
//
//	reused across all fence terms since each is consumed immediately.
//
// delta:        the channel amount to shift when fully past this fence
//
//	(e.g. -35 for R at fence 1 in our example)
//
// sigma:        half-width of the blur zone (e.g. 0.05)
func buildSmoothstepTerm(boundarySlot, tempSlot, delta int, sigma float64) string {
	twoSigma := 2 * sigma

	return fmt.Sprintf(
		"%+d*st(%d,min(1,max(0,(ld(0)-ld(%d)+%f)/%f)))*ld(%d)*(3-2*ld(%d))",
		delta,
		tempSlot,
		boundarySlot,
		sigma,
		twoSigma,
		tempSlot,
		tempSlot,
	)
}

// buildChannelExpr assembles the full geq expression for one color channel.
// This is the string that goes inside r='...' or g='...' or b='...'.
//
// Structure of the output:
//
//	st(0, shape position);             ← store s (see shapeExpr)
//	st(1, T/duration);                 ← store progress p
//	st(2, intercept_1 + slope_1*ld(1)); ← store fence 1 position
//	st(3, intercept_2 + slope_2*ld(1)); ← store fence 2 position
//	... one st() per fence ...
//	baseVal                            ← start from innermost color's channel value
//	+ term_for_fence_1                 ← each term from buildSmoothstepTerm
//	+ term_for_fence_2
//	...
//
// shape:    geq expression for the position s (0..1) of the current pixel
// spec:     the color bands. Base[channel] is the innermost color's value (e.g.
// 255 for R of FFF8EC); Deltas[k][channel] is the shift at fence k (e.g.
// -35, -67, -69 for R).
// channel:  0=R, 1=G, 2=B
// duration: clip length in seconds
func buildChannelExpr(shape string, spec gradient.Bands, channel, duration int) string {
	var expr strings.Builder

	// Store the pixel's position (normalized 0 -> 1)
	fmt.Fprintf(&expr, "st(0,%s);", shape)

	// Store progress through clip (0 -> 1)
	fmt.Fprintf(&expr, "st(1,T/%d.0);", duration)
	// t = clamp((d - fence + sigma) / (2*sigma), 0, 1)
	// s = t*t*(3-2*t)
	// contribution = delta * s
	//
	// st(tempSlot, t) stores t AND returns it, so we can reuse it in the smoothstep formula
	// ld(0) = s (position of the pixel)
	// ld(boundarySlot) = fence position at current time

	// Store each fence position at current time
	// Slot 2 = fence 1, slot 3 = fence 2, etc.
	for i, fence := range spec.Fences {
		slot := i + 2
		fmt.Fprintf(&expr, "st(%d,%f+%f*ld(1));", slot, fence.Intercept, fence.Slope)
	}

	// Start from innermost color's channel value
	fmt.Fprintf(&expr, "%d", spec.Base[channel])

	// Add each fence's contribution
	// Fence i is in slot i+2, temp smoothstep value goes in the first slot
	// past the last fence slot (reused by every term)
	tempSlot := len(spec.Fences) + 2
	for i := range spec.Fences {
		boundarySlot := i + 2
		delta := int(spec.Deltas[i][channel])
		expr.WriteString(buildSmoothstepTerm(boundarySlot, tempSlot, delta, spec.Sigma))
	}

	return expr.String()
}

// shapeExpr returns the geq expression that evaluates the gradient shape at
// the current pixel, mirroring gradient.RadialShape / gradient.LinearShape.
// X, Y, W, H are geq's built-in pixel coordinates and frame size.
func shapeExpr(cfg config.VideoConfig) (string, error) {
	switch cfg.GradientType {
	case config.GradientRadial:
		return "hypot(X-W/2,Y-H/2)/hypot(W/2,H/2)", nil
	case config.GradientLinear:
		// Parenthesized constants: they may be negative or -0.
		ux, uy := gradient.Direction(cfg.LinearAngle)

		return fmt.Sprintf(
			"1-(0.5+((X-W/2)*(%f)+(Y-H/2)*(%f))/(abs(W*(%f))+abs(H*(%f))))",
			ux, uy, ux, uy,
		), nil
	default:
		return "", fmt.Errorf("unsupported gradient type %q", cfg.GradientType)
	}
}

// Build is the entry point. Reads gradient type, colors, angle and duration
// from config, runs all the math, and returns the complete geq filter string
// for ffmpeg's -vf flag.
//
// Builds a gradient.Bands spec from the palette, then buildChannelExpr
// three times (once per channel), and joins them as:
//
//	"geq=r='...':g='...':b='...'"
//
// Returns an error if the gradient type is unsupported, any color string is
// malformed or the Colors slice is too short.
func Build(cfg config.VideoConfig) (string, error) {
	spec, err := gradient.NewBands(cfg.Colors)
	if err != nil {
		return "", err
	}

	shape, err := shapeExpr(cfg)
	if err != nil {
		return "", err
	}

	rExpr := buildChannelExpr(shape, spec, 0, cfg.Duration)
	gExpr := buildChannelExpr(shape, spec, 1, cfg.Duration)
	bExpr := buildChannelExpr(shape, spec, 2, cfg.Duration)

	return fmt.Sprintf("geq=r='%s':g='%s':b='%s'", rExpr, gExpr, bExpr), nil
}
