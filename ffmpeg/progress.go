package ffmpeg

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// progress counts rendered frames and prints a one-line status to stderr
// while running. Printing is skipped when stderr is not a terminal.
type progress struct {
	total int
	done  atomic.Int64
}

func newProgress(total int) *progress {
	return &progress{total: total}
}

// Tick records one finished frame. Safe for concurrent use.
func (p *progress) Tick() { p.done.Add(1) }

// start begins periodic printing and returns a function that stops it and
// prints the final line.
func (p *progress) start() (stop func()) {
	info, err := os.Stderr.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return func() {}
	}

	begin := time.Now()
	quit := make(chan struct{})
	finished := make(chan struct{})

	emit := func(final bool) {
		done := int(p.done.Load())
		elapsed := time.Since(begin).Seconds()

		fps := 0.0
		if elapsed > 0 {
			fps = float64(done) / elapsed
		}

		eta := "?"
		if fps > 0 {
			eta = (time.Duration(float64(p.total-done)/fps) * time.Second).Round(time.Second).String()
		}

		end := "\r"
		if final {
			end = "\n"
		}

		fmt.Fprintf(os.Stderr, "\rframes %d/%d  %.1f fps  eta %s\033[K%s", done, p.total, fps, eta, end)
	}

	go func() {
		defer close(finished)

		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				emit(false)
			case <-quit:
				emit(true)
				return
			}
		}
	}()

	return func() {
		close(quit)
		<-finished
	}
}
