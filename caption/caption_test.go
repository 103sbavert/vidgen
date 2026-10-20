package caption

import (
	"testing"
)

func mustLayout(t *testing.T, lines []string, size float64, w, h int) *Mask {
	t.Helper()

	m, err := Layout(lines, size, w, h)
	if err != nil {
		t.Fatal(err)
	}

	return m
}

func inkSum(m *Mask) (sum int) {
	for _, a := range m.Alpha {
		sum += int(a)
	}

	return sum
}

func TestLayoutCentersText(t *testing.T) {
	const w, h = 640, 360

	m := mustLayout(t, []string{"Hello"}, 48, w, h)
	if m.Empty() {
		t.Fatal("no ink")
	}

	if len(m.Alpha) != m.W*m.H {
		t.Fatalf("alpha has %d values for %dx%d", len(m.Alpha), m.W, m.H)
	}

	// Ink box center within a few pixels of the frame center (glyph shapes and
	// ascender/descender balance keep it from being exact).
	cx, cy := m.X+m.W/2, m.Y+m.H/2
	if abs(cx-w/2) > 6 || abs(cy-h/2) > 14 {
		t.Errorf("ink center (%d,%d), want near (%d,%d)", cx, cy, w/2, h/2)
	}

	if m.X < 0 || m.Y < 0 || m.X+m.W > w || m.Y+m.H > h {
		t.Errorf("mask %+v outside the frame", *m)
	}
}

func TestLayoutTightBoundingBox(t *testing.T) {
	m := mustLayout(t, []string{"Hello"}, 48, 640, 360)

	rowHasInk := func(y int) bool {
		for x := range m.W {
			if m.Alpha[y*m.W+x] != 0 {
				return true
			}
		}

		return false
	}

	colHasInk := func(x int) bool {
		for y := range m.H {
			if m.Alpha[y*m.W+x] != 0 {
				return true
			}
		}

		return false
	}

	if !rowHasInk(0) || !rowHasInk(m.H-1) || !colHasInk(0) || !colHasInk(m.W-1) {
		t.Errorf("mask %dx%d has an inkless edge: box is not tight", m.W, m.H)
	}
}

func TestLayoutScalesWithSize(t *testing.T) {
	small := mustLayout(t, []string{"Hello"}, 24, 640, 360)
	big := mustLayout(t, []string{"Hello"}, 48, 640, 360)

	if big.W < small.W*3/2 || big.H < small.H*3/2 {
		t.Errorf("48px mask %dx%d should be about twice the 24px mask %dx%d", big.W, big.H, small.W, small.H)
	}
}

func TestLayoutMultipleLines(t *testing.T) {
	one := mustLayout(t, []string{"Hello"}, 32, 640, 360)
	three := mustLayout(t, []string{"Hello", "Hello", "Hello"}, 32, 640, 360)

	if three.H < one.H*2 {
		t.Errorf("three lines are %d tall, one line is %d", three.H, one.H)
	}

	// Identical lines: three times the ink.
	if got, want := inkSum(three), 3*inkSum(one); got < want*95/100 || got > want*105/100 {
		t.Errorf("three lines have ink %d, want about %d", got, want)
	}
}

func TestLayoutBlank(t *testing.T) {
	for name, lines := range map[string][]string{
		"no lines":   nil,
		"empty line": {""},
		"spaces":     {"   "},
	} {
		if m := mustLayout(t, lines, 32, 320, 240); !m.Empty() {
			t.Errorf("%s: mask %+v should be empty", name, *m)
		}
	}
}

func TestLayoutClipsOversizedText(t *testing.T) {
	m := mustLayout(t, []string{"This line is far too wide for the frame"}, 64, 200, 100)
	if m.Empty() {
		t.Fatal("clipped text should still show")
	}

	if m.X < 0 || m.Y < 0 || m.X+m.W > 200 || m.Y+m.H > 100 {
		t.Errorf("mask %+v outside the 200x100 frame", *m)
	}
}

func TestLayoutRejectsBadInput(t *testing.T) {
	for name, call := range map[string]func() error{
		"zero size":     func() error { _, err := Layout([]string{"x"}, 0, 100, 100); return err },
		"negative size": func() error { _, err := Layout([]string{"x"}, -3, 100, 100); return err },
		"zero width":    func() error { _, err := Layout([]string{"x"}, 20, 0, 100); return err },
		"zero height":   func() error { _, err := Layout([]string{"x"}, 20, 100, 0); return err },
	} {
		if call() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestNRGBA(t *testing.T) {
	const w, h = 320, 240

	m := mustLayout(t, []string{"Hi"}, 40, w, h)
	img := m.NRGBA(w, h, [3]uint8{10, 20, 30})

	if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		t.Fatalf("image is %v, want %dx%d", img.Bounds(), w, h)
	}

	inked := 0

	for y := range h {
		for x := range w {
			c := img.NRGBAAt(x, y)

			inside := x >= m.X && x < m.X+m.W && y >= m.Y && y < m.Y+m.H
			if !inside {
				if c.A != 0 {
					t.Fatalf("pixel (%d,%d) outside the mask is not transparent", x, y)
				}

				continue
			}

			if want := m.Alpha[(y-m.Y)*m.W+(x-m.X)]; c.A != want {
				t.Fatalf("pixel (%d,%d) alpha %d, want %d", x, y, c.A, want)
			}

			if c.A != 0 {
				inked++

				if c.R != 10 || c.G != 20 || c.B != 30 {
					t.Fatalf("pixel (%d,%d) color %v", x, y, c)
				}
			}
		}
	}

	if inked == 0 {
		t.Error("no inked pixels")
	}

	if empty := (&Mask{}).NRGBA(w, h, [3]uint8{}); empty.Bounds().Dx() != w {
		t.Error("empty mask must still give a full-size transparent image")
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}
