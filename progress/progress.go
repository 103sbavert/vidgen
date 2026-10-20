// Package progress prints a one-line frame counter, fps and ETA while a render
// runs. It only consumes Tick calls, so rendering code stays free of terminal
// concerns.
package progress

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

const interval = 200 * time.Millisecond

// Reporter counts finished frames and periodically prints a status line.
type Reporter struct {
	w     io.Writer
	total int
	done  atomic.Int64
}

// New returns a Reporter for a job of total frames that prints to w.
// Nothing is printed unless w is a terminal.
func New(w io.Writer, total int) *Reporter {
	return &Reporter{w: w, total: total}
}

// Tick records one finished frame. Safe for concurrent use.
func (r *Reporter) Tick() { r.done.Add(1) }

// Start begins periodic printing. The returned function stops it and prints
// the final line; call it exactly once.
func (r *Reporter) Start() (stop func()) {
	if !isTerminal(r.w) {
		return func() {}
	}

	begin := time.Now()
	quit := make(chan struct{})
	finished := make(chan struct{})

	emit := func(end string) {
		fmt.Fprintf(r.w, "\r%s\033[K%s", Line(int(r.done.Load()), r.total, time.Since(begin)), end)
	}

	go func() {
		defer close(finished)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				emit("")
			case <-quit:
				emit("\n")
				return
			}
		}
	}()

	return func() {
		close(quit)
		<-finished
	}
}

// Line formats the status for done of total frames after elapsed time.
func Line(done, total int, elapsed time.Duration) string {
	fps := 0.0
	if elapsed > 0 {
		fps = float64(done) / elapsed.Seconds()
	}

	eta := "?"
	if fps > 0 {
		remaining := float64(max(0, total-done)) / fps
		eta = time.Duration(remaining * float64(time.Second)).Round(time.Second).String()
	}

	return fmt.Sprintf("frames %d/%d  %.1f fps  eta %s", done, total, fps, eta)
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
