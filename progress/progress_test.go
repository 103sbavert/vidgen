package progress

import (
	"bytes"
	"testing"
	"time"
)

func TestLine(t *testing.T) {
	got := Line(50, 150, 5*time.Second)
	want := "frames 50/150  10.0 fps  eta 10s"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := Line(0, 100, 0); got != "frames 0/100  0.0 fps  eta ?" {
		t.Errorf("no data: %q", got)
	}
}

func TestStartIsSilentOffTerminal(t *testing.T) {
	var buf bytes.Buffer

	r := New(&buf, 10)
	stop := r.Start()
	r.Tick()
	stop()

	if buf.Len() != 0 {
		t.Errorf("printed to non-terminal: %q", buf.String())
	}
}
