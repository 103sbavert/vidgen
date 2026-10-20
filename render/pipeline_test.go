package render

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// countSource writes the frame number (as a single repeated byte pair) so
// ordering is easy to verify.
type countSource struct {
	frames int
}

func (countSource) Layout() Layout { return NewLayout(4, 2) }
func (s countSource) Frames() int  { return s.frames }

func (countSource) Render(n int, buf []byte) {
	// Jitter so workers finish out of order.
	time.Sleep(time.Duration((n*7919)%5) * 100 * time.Microsecond)

	for i := range buf {
		buf[i] = byte(n)
	}
}

func TestStreamOrder(t *testing.T) {
	src := countSource{frames: 200}
	size := src.Layout().FrameSize()

	for _, workers := range []int{1, 3, 16} {
		var out bytes.Buffer

		var ticks int

		err := Stream(context.Background(), src, 10, 190, &out, Options{
			Workers:  workers,
			Progress: func() { ticks++ },
		})
		if err != nil {
			t.Fatalf("workers=%d: %v", workers, err)
		}

		if ticks != 180 {
			t.Errorf("workers=%d: %d progress ticks, want 180", workers, ticks)
		}

		got := out.Bytes()
		if len(got) != 180*size {
			t.Fatalf("workers=%d: wrote %d bytes, want %d", workers, len(got), 180*size)
		}

		for i := range 180 {
			if got[i*size] != byte(10+i) {
				t.Fatalf("workers=%d: frame slot %d holds frame %d, want %d", workers, i, got[i*size], 10+i)
			}
		}
	}
}

type failWriter struct{ after int }

func (w *failWriter) Write(p []byte) (int, error) {
	if w.after--; w.after < 0 {
		return 0, io.ErrClosedPipe
	}

	return len(p), nil
}

func TestStreamWriterErrorDoesNotHang(t *testing.T) {
	done := make(chan error, 1)

	go func() {
		done <- Stream(context.Background(), countSource{frames: 500}, 0, 500, &failWriter{after: 7}, Options{Workers: 8})
	}()

	select {
	case err := <-done:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("got %v, want io.ErrClosedPipe", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stream hung after writer error")
	}
}

func TestStreamContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Stream(ctx, countSource{frames: 100}, 0, 100, io.Discard, Options{Workers: 4})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}
