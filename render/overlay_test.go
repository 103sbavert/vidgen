package render

import (
	"testing"

	"sbavert/vidgen/caption"
)

// flatFrame returns a frame whose planes are all filled with fixed values.
func flatFrame(l Layout, y, u, v uint8) []byte {
	buf := make([]byte, l.FrameSize())
	yp, up, vp := l.Planes(buf)

	for i := range yp {
		yp[i] = y
	}

	for i := range up {
		up[i], vp[i] = u, v
	}

	return buf
}

func solidMask(x, y, w, h int, alpha uint8) *caption.Mask {
	m := &caption.Mask{X: x, Y: y, W: w, H: h, Alpha: make([]uint8, w*h)}
	for i := range m.Alpha {
		m.Alpha[i] = alpha
	}

	return m
}

func TestOverlayFullCoverageSetsTextColor(t *testing.T) {
	l := NewLayout(64, 48)
	buf := flatFrame(l, 100, 128, 128)

	o := NewOverlay(l, solidMask(10, 8, 20, 16, 255), [3]uint8{255, 255, 255})
	o.Apply(buf)

	yp, up, vp := l.Planes(buf)
	wantY, wantU, wantV := yuv709(255, 255, 255)

	if got := yp[10*l.W+15]; got != uint8(wantY+0.5) {
		t.Errorf("inside luma = %d, want %d", got, uint8(wantY+0.5))
	}

	if got := up[6*l.CW+10]; got != uint8(wantU+0.5) || vp[6*l.CW+10] != uint8(wantV+0.5) {
		t.Errorf("inside chroma = %d,%d want %d,%d", got, vp[6*l.CW+10], uint8(wantU+0.5), uint8(wantV+0.5))
	}
}

func TestOverlayLeavesOutsideUntouched(t *testing.T) {
	l := NewLayout(64, 48)
	buf := flatFrame(l, 100, 90, 160)
	want := append([]byte(nil), buf...)

	NewOverlay(l, solidMask(10, 8, 20, 16, 255), [3]uint8{0, 0, 0}).Apply(buf)

	yp, up, _ := l.Planes(buf)
	wp, wu, _ := l.Planes(want)

	for y := range l.H {
		for x := range l.W {
			inside := x >= 10 && x < 30 && y >= 8 && y < 24
			if !inside && yp[y*l.W+x] != wp[y*l.W+x] {
				t.Fatalf("luma changed outside the mask at (%d,%d)", x, y)
			}
		}
	}

	// Chroma sample (cx,cy) covers luma (2cx..2cx+1, 2cy..2cy+1): the mask is
	// even-aligned here, so the changed chroma block is exactly 5..14 x 4..11.
	for cy := range l.CH {
		for cx := range l.CW {
			inside := cx >= 5 && cx < 15 && cy >= 4 && cy < 12
			if !inside && up[cy*l.CW+cx] != wu[cy*l.CW+cx] {
				t.Fatalf("chroma changed outside the mask at (%d,%d)", cx, cy)
			}
		}
	}
}

func TestOverlayPartialCoverageBlends(t *testing.T) {
	l := NewLayout(32, 32)
	buf := flatFrame(l, 200, 128, 128)

	// Text luma will be 16 (black); half coverage lands midway.
	NewOverlay(l, solidMask(4, 4, 8, 8, 128), [3]uint8{0, 0, 0}).Apply(buf)

	yp, _, _ := l.Planes(buf)

	if got := yp[6*l.W+6]; got < 100 || got > 112 {
		t.Errorf("half-covered luma = %d, want about 108", got)
	}
}

func TestOverlayZeroCoverageIsNoOp(t *testing.T) {
	l := NewLayout(32, 32)
	buf := flatFrame(l, 77, 100, 150)
	want := append([]byte(nil), buf...)

	NewOverlay(l, solidMask(4, 4, 8, 8, 0), [3]uint8{255, 0, 0}).Apply(buf)

	for i := range buf {
		if buf[i] != want[i] {
			t.Fatalf("byte %d changed with zero coverage", i)
		}
	}
}

// A mask at odd coordinates and a frame of odd size must not panic or write
// out of bounds, including the last column and row.
func TestOverlayOddGeometry(t *testing.T) {
	l := NewLayout(33, 25)
	buf := flatFrame(l, 100, 128, 128)

	o := NewOverlay(l, solidMask(27, 19, 6, 6, 255), [3]uint8{255, 255, 255})
	o.Apply(buf)

	yp, _, _ := l.Planes(buf)
	if yp[24*l.W+32] <= 100 {
		t.Error("bottom-right pixel was not drawn")
	}
}

func TestNewOverlayEmptyMask(t *testing.T) {
	if NewOverlay(NewLayout(32, 32), &caption.Mask{}, [3]uint8{}) != nil {
		t.Error("empty mask should give a nil overlay")
	}

	if NewOverlay(NewLayout(32, 32), nil, [3]uint8{}) != nil {
		t.Error("nil mask should give a nil overlay")
	}
}

type stubSource struct{ layout Layout }

func (s stubSource) Layout() Layout { return s.layout }
func (stubSource) Frames() int      { return 3 }

func (s stubSource) Render(n int, buf []byte) {
	copy(buf, flatFrame(s.layout, 100, 128, 128))
}

func TestWithOverlay(t *testing.T) {
	l := NewLayout(32, 32)
	src := stubSource{layout: l}

	if got := WithOverlay(src, nil); got != Source(src) {
		t.Error("nil overlay must return the source unchanged")
	}

	wrapped := WithOverlay(src, NewOverlay(l, solidMask(4, 4, 8, 8, 255), [3]uint8{255, 255, 255}))

	if wrapped.Frames() != 3 || wrapped.Layout() != l {
		t.Error("wrapper must forward Layout and Frames")
	}

	buf := make([]byte, l.FrameSize())
	wrapped.Render(0, buf)

	yp, _, _ := l.Planes(buf)
	if yp[6*l.W+6] == 100 || yp[0] != 100 {
		t.Errorf("overlay not applied inside only: inside=%d outside=%d", yp[6*l.W+6], yp[0])
	}
}
