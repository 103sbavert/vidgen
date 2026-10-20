// Package gradient holds the engine-independent math of the animated
// banded gradients: band geometry, fence motion and color deltas. Both the
// native renderer (package render) and the geq expression builder (package
// ffmpeg/geq) are driven from the same Bands spec and Shape, so they stay in
// sync.
//
// A gradient is two independent parts:
//   - Shape: maps a pixel to a scalar position (s) in 0→1. Radial shape: distance
//     from the center. Linear shape: distance along a direction.
//   - Bands: maps (s, progress) to a color. Rings/stripes start at unequal
//     widths and animate toward equal width over the clip.
//
// Key concepts:
//   - Position (s): normalized 0→1 from the innermost color to the outermost
//   - Progress (p): normalized 0→1 from clip start to end
//   - Fences: dividing lines between bands, moving linearly with p
//   - Smoothstep: S-curve blending of colors across a fence
//   - Color deltas: RGB shift applied when crossing each fence
package gradient

import (
	"fmt"

	"sbavert/vidgen/config"
)

// Sigma is the half-width of the fuzzy transition zone around each fence.
const Sigma = 0.05

// Fence describes how one band boundary moves: position = Intercept + Slope*p.
type Fence struct {
	Intercept float64
	Slope     float64
}

// At returns the fence position at progress p (0→1).
func (f Fence) At(p float64) float64 {
	return f.Intercept + f.Slope*p
}

// Bands is the complete color description of an animated banded gradient.
//
// Colors in config are ordered outermost-first (s=1 is the first color, s=0
// the last). Fences and Deltas are ordered innermost-first: Fences[i] sits
// between band i and band i+1 counted from s=0, and Deltas[i] is the RGB
// shift applied when crossing it toward larger s. The color at position s and
// progress p is
//
//	Base + sum_i Deltas[i] * smoothstep((s - Fences[i].At(p) + Sigma) / (2*Sigma))
type Bands struct {
	Base   [3]uint8
	Deltas [][3]int16
	Fences []Fence
	Sigma  float64
}

// Smoothstep returns the S-curve blend factor (0 inside the fence, 1
// outside) for a pixel at normalized position d against a fence at position
// pos with transition half-width sigma.
func Smoothstep(d, pos, sigma float64) float64 {
	t := min(1, max(0, (d-pos+sigma)/(2*sigma)))

	return t * t * (3 - 2*t)
}

// ColorAt evaluates the gradient directly: the RGB color (0..255, unrounded)
// at normalized position d and progress p. This is the canonical definition;
// the geq expression and the native renderer's tables must agree with it.
func (g Bands) ColorAt(d, p float64) (rgb [3]float64) {
	for c := range rgb {
		rgb[c] = float64(g.Base[c])
	}

	for k, fence := range g.Fences {
		s := Smoothstep(d, fence.At(p), g.Sigma)

		for c := range rgb {
			rgb[c] += float64(g.Deltas[k][c]) * s
		}
	}

	for c := range rgb {
		rgb[c] = min(255, max(0, rgb[c]))
	}

	return rgb
}

// BoundaryConstants returns the intercept and slope that fully describe how
// boundary k animates over time. Boundaries are numbered 1 through N-1,
// from inside outward; boundary k sits between the k-th and (k+1)-th ring
// from the center.
//
// intercept = fence position at p=0
//
//	= sum of initial widths of the k innermost rings
//	= k*(k+1) / (N*(N+1))   ← closed-form, no loop needed
//
// target = fence position at p=1 (all rings equal width)
//
//	= k / N
//
// slope = target - intercept (total distance the fence travels)
//
// For N=4, the 3 fences are:
//
//	k=1: intercept=0.10, target=0.25, slope=+0.15
//	k=2: intercept=0.30, target=0.50, slope=+0.20
//	k=3: intercept=0.60, target=0.75, slope=+0.15
func BoundaryConstants(k, N int) (intercept, slope float64) {
	boundaryIndex := float64(k)
	totalRings := float64(N)

	intercept = boundaryIndex * (boundaryIndex + 1) / (totalRings * (totalRings + 1))
	target := boundaryIndex / totalRings
	slope = target - intercept

	return intercept, slope
}

// ColorDeltas computes, for each fence between adjacent rings, the RGB shift
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
// Index 0=R, 1=G, 2=B, innermost fence first. Parse errors for individual
// colors are collected and returned alongside any computed deltas.
func ColorDeltas(colors []config.Color) (deltas [][3]int16, errorList []error) {
	lastIndex := len(colors) - 1

	for i := lastIndex - 1; i >= 0; i-- {
		innerColor := colors[i+1]
		outerColor := colors[i]

		innerR, innerG, innerB, errInner := innerColor.ParseHex()
		outerR, outerG, outerB, errOuter := outerColor.ParseHex()

		if errInner != nil {
			errorList = append(errorList, fmt.Errorf("could not parse band color %s at position %d: %w", innerColor, i+1, errInner))
		}

		if errOuter != nil {
			errorList = append(errorList, fmt.Errorf("could not parse band color %s at position %d: %w", outerColor, i, errOuter))
		}

		if errInner != nil || errOuter != nil {
			continue
		}

		deltas = append(deltas, [3]int16{
			int16(outerR) - int16(innerR),
			int16(outerG) - int16(innerG),
			int16(outerB) - int16(innerB),
		})
	}

	return
}

// NewBands builds the color description for colors (outermost-first).
// Returns an error if there are fewer than 2 colors or any color is malformed.
func NewBands(colors []config.Color) (Bands, error) {
	N := len(colors)

	if N == 0 {
		return Bands{}, fmt.Errorf("no colors provided")
	}

	if N == 1 {
		return Bands{}, fmt.Errorf("need at least 2 colors for a gradient")
	}

	innermost := colors[N-1]
	r, g, b, err := innermost.ParseHex()
	if err != nil {
		return Bands{}, fmt.Errorf("invalid innermost color %s: %w", innermost, err)
	}

	deltas, errs := ColorDeltas(colors)
	if len(errs) > 0 {
		return Bands{}, fmt.Errorf("color parsing errors: %v", errs)
	}

	if len(deltas) != N-1 {
		return Bands{}, fmt.Errorf("expected %d deltas, got %d", N-1, len(deltas))
	}

	fences := make([]Fence, 0, N-1)
	for k := 1; k < N; k++ {
		intercept, slope := BoundaryConstants(k, N)
		fences = append(fences, Fence{Intercept: intercept, Slope: slope})
	}

	return Bands{
		Base:   [3]uint8{r, g, b},
		Deltas: deltas,
		Fences: fences,
		Sigma:  Sigma,
	}, nil
}
