package render

import (
	"context"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
)

// Source produces frames. Render must be safe for concurrent calls.
type Source interface {
	Layout() Layout
	Frames() int
	Render(n int, buf []byte)
}

// Options tunes Stream.
type Options struct {
	// Workers is the number of render goroutines; <=0 means runtime.NumCPU().
	Workers int
	// Progress, if set, is called after each frame is written. It is called
	// from the writing goroutine only.
	Progress func()
}

type slot struct {
	buf   []byte
	ready chan struct{} // worker -> writer: buf holds the frame
	free  chan struct{} // writer -> worker: buf may be overwritten
}

// Stream renders frames [start, end) of src with a pool of workers and
// writes them to w strictly in order.
//
// Frames live in a ring of reusable buffers. Frame i uses slot i%R and may
// only be rendered once frame i-R has been written, which bounds memory and
// lets rendering run ahead of a slow consumer without ever deadlocking
// (workers claim frame indices in increasing order, so the frame the writer
// is waiting for is always held by a running worker).
func Stream(ctx context.Context, src Source, start, end int, w io.Writer, opt Options) error {
	total := end - start
	if total <= 0 {
		return nil
	}

	workers := opt.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	workers = min(workers, total)
	ringSize := min(workers+4, total)
	frameSize := src.Layout().FrameSize()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	slots := make([]slot, ringSize)
	for i := range slots {
		slots[i] = slot{
			buf:   make([]byte, frameSize),
			ready: make(chan struct{}, 1),
			free:  make(chan struct{}, 1),
		}
		slots[i].free <- struct{}{}
	}

	var (
		next atomic.Int64
		wg   sync.WaitGroup
	)

	for range workers {
		wg.Go(func() {
			for {
				i := int(next.Add(1) - 1)
				if i >= total {
					return
				}

				s := &slots[i%ringSize]

				select {
				case <-s.free:
				case <-ctx.Done():
					return
				}

				src.Render(start+i, s.buf)

				select {
				case s.ready <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	var err error

	for i := range total {
		s := &slots[i%ringSize]

		select {
		case <-s.ready:
		case <-ctx.Done():
			err = ctx.Err()
		}

		if err != nil {
			break
		}

		if _, err = w.Write(s.buf); err != nil {
			break
		}

		s.free <- struct{}{}

		if opt.Progress != nil {
			opt.Progress()
		}
	}

	cancel()
	wg.Wait()

	return err
}
