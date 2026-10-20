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
//	st(0, distance from center);       ← store d
//	st(1, T/duration);                 ← store progress p
//	st(2, intercept_1 + slope_1*ld(1)); ← store fence 1 position
//	st(3, intercept_2 + slope_2*ld(1)); ← store fence 2 position
//	... one st() per fence ...
//	baseVal                            ← start from innermost color's channel value
//	+ term_for_fence_1                 ← each term from buildSmoothstepTerm
//	+ term_for_fence_2
//	...
//
// baseVal:  the innermost color's value for this channel (e.g. 255 for R of FFF8EC)
// deltas:   per-fence shift for this channel (e.g. [-35, -67, -69] for R)
// N:        total number of rings/colors
// sigma:    blur half-width
// duration: clip length in seconds
func buildChannelExpr(baseVal int, deltas []int16, N int, sigma float64, duration int) string {
	var expr strings.Builder

	// Store distance from center (normalized 0 -> 1)
	expr.WriteString("st(0,hypot(X-W/2,Y-H/2)/hypot(W/2,H/2));")

	// Store progress through clip (0 -> 1)
	fmt.Fprintf(&expr, "st(1,T/%d.0);", duration)
	// t = clamp((d - fence + sigma) / (2*sigma), 0, 1)
	// s = t*t*(3-2*t)
	// contribution = delta * s
	//
	// st(tempSlot, t) stores t AND returns it, so we can reuse it in the smoothstep formula
	// ld(0) = d (distance from center)
	// ld(boundarySlot) = fence position at current time

	// Store each fence position at current time
	// Slot 2 = fence 1, slot 3 = fence 2, etc.
	for k := 1; k < N; k++ {
		intercept, slope := gradient.BoundaryConstants(k, N)
		slot := k + 1
		fmt.Fprintf(&expr, "st(%d,%f+%f*ld(1));", slot, intercept, slope)
	}

	// Start from innermost color's channel value
	fmt.Fprintf(&expr, "%d", baseVal)

	// Add each fence's contribution
	// Fence k is in slot k+1, temp smoothstep value goes in slot N+1 (reused,
	// always past the last fence slot N)
	for k := 1; k < N; k++ {
		boundarySlot := k + 1
		tempSlot := N + 1
		delta := int(deltas[k-1])
		expr.WriteString(buildSmoothstepTerm(boundarySlot, tempSlot, delta, sigma))
	}

	return expr.String()
}

// BuildRadialGeq is the entry point. Reads colors and duration from config,
// runs all the math, and returns the complete geq filter string for ffmpeg's -vf flag.
//
// Builds a gradient.Radial spec from the palette, then buildChannelExpr
// three times (once per channel), and joins them as:
//
//	"geq=r='...':g='...':b='..."
//
// Returns an error if any color string is malformed or Colors slice is empty.
func BuildRadialGeq(cfg config.VideoConfig) (string, error) {
	spec, err := gradient.NewRadial(cfg.Colors)
	if err != nil {
		return "", err
	}

	N := len(cfg.Colors)

	// Extract per-channel deltas
	var deltaR, deltaG, deltaB []int16
	for _, d := range spec.Deltas {
		deltaR = append(deltaR, d[0])
		deltaG = append(deltaG, d[1])
		deltaB = append(deltaB, d[2])
	}

	// Build each channel expression
	rExpr := buildChannelExpr(int(spec.Base[0]), deltaR, N, spec.Sigma, cfg.Duration)
	gExpr := buildChannelExpr(int(spec.Base[1]), deltaG, N, spec.Sigma, cfg.Duration)
	bExpr := buildChannelExpr(int(spec.Base[2]), deltaB, N, spec.Sigma, cfg.Duration)

	return fmt.Sprintf("geq=r='%s':g='%s':b='%s'", rExpr, gExpr, bExpr), nil
}
