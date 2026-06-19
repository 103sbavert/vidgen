package geq

import (
	"fmt"
	"sbavert/vidgen/config"
)

// ringWidth returns the initial fractional width of one ring on the 0→1 radius ruler.
// Colors in config are ordered outermost-first, so n=0 is the widest ring.
//
// Formula: 2*(N-n) / (N*(N+1))
//
// For the 4 given colors (N=4):
//
//	n=0 → 2*4/20 = 0.40  (546B41,  outermost, widest)
//	n=1 → 2*3/20 = 0.30  (99AD7A)
//	n=2 → 2*2/20 = 0.20  (DCCCAC)
//	n=3 → 2*1/20 = 0.10  (FFF8EC, innermost, narrowest)
//
// These sum to exactly 1.0 for any N, so the rings always fill the full radius.
func ringInitialWidth(n, N int) float64 {
	ringIndex := float64(n)
	totalRings := float64(N)

	return 2 * (totalRings - ringIndex) / (totalRings * (totalRings + 1))
}

// boundaryConstants returns the intercept and slope that fully describe how
// boundary k animates over time. Boundaries are numbered 1 through N-1,
// from inside outward; boundary k sits between the k-th and (k+1)-th ring
// from the center.
//
// intercept = fence position at T=0
//
//	= sum of initial widths of the k innermost rings
//	= k*(k+1) / (N*(N+1))   ← closed-form, no loop needed
//
// target = fence position at T=end (all rings equal width)
//
//	= k / N
//
// slope = target - intercept (total distance the fence travels)
//
// At any time T: fence position = intercept + slope*(T/duration)
//
// For N=4, the 3 fences are:
//
//	k=1: intercept=0.10, target=0.25, slope=+0.15
//	k=2: intercept=0.30, target=0.50, slope=+0.20
//	k=3: intercept=0.60, target=0.75, slope=+0.15

func boundaryConstants(k, N int) (intercept, slope float64) {
	boundaryIndex := float64(k)
	totalRings := float64(N)

	intercept = boundaryIndex * (boundaryIndex + 1) / (totalRings * (totalRings + 1))
	target := boundaryIndex / totalRings
	slope = target - intercept

	return intercept, slope
}

// colorDeltas computes, for each fence between adjacent rings, the RGB shift
// applied when crossing that fence. Colors in config are outermost-first;
// fences are innermost-first, so the delta at fence k is color[N-1-k] minus
// color[N-k] (one step outward minus one step inward).
//
// For N=4:
//
//	fence 1 (innermost): DCCCAC - FFF8EC = (-35, -44, -64)
//	fence 2 (middle):    99AD7A - DCCCAC = (-67, -31, -50)
//	fence 3 (outermost): 546B41 - 99AD7A = (-69, -66, -57)
//
// Deltas may be positive, negative, or mixed depending on input colors.
// Returns a slice of [3]int16 (index 0=R, 1=G, 2=B), ordered innermost
// fence first. Parse errors for individual colors are collected and returned
// alongside any successfully computed deltas.
func colorDeltas(colors []config.Color) (deltas [][3]int16, errorList []error) {

	lastIndex := len(colors) - 1

	for i := lastIndex - 1; i >= 0; i-- {
		innerColor := colors[i+1]
		outerColor := colors[i]

		innerR, innerG, innerB, errInner := innerColor.ParseHex()
		outerR, outerG, outerB, errOuter := outerColor.ParseHex()

		if errInner != nil {
			errorList = append(errorList, fmt.Errorf("coud not parse ring color %s at position %d: %w", innerColor, i+1, errInner))

		}

		if errOuter != nil {
			errorList = append(errorList, fmt.Errorf("coud not parse ring color %s at position %d: %w", outerColor, i, errOuter))
		}

		if errInner != nil || errOuter != nil {
			continue
		}

		currDelta := [3]int16{int16(outerR) - int16(innerR), int16(outerG) - int16(innerG), int16(outerB) - int16(innerB)}

		deltas = append(deltas, currDelta)
	}

	return deltas, errorList
}

// Builds one geq string fragment representing a single fence's color contribution
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
func buildSmoothstepTerm(boundarySlot, tempSlot, delta int, sigma float64) string
