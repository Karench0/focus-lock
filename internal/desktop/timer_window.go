package desktop

import (
	"math"
	"sync"
	"time"
)

const CompactWidth, CompactHeight = 420, 180

type Bounds struct{ X, Y, Width, Height int }

// Window keeps native presentation separate from the authoritative session.
type Window interface {
	Bounds() Bounds
	SetSize(int, int)
	SetPosition(int, int)
	SetMinSize(int, int)
	SetMaxSize(int, int)
	SetAlwaysOnTop(bool)
	IsMaximised() bool
	Unmaximise()
	Maximise()
}

// TimerWindow serializes transitions, including restore requests from the tray.
// The saved normal bounds survive dragging the compact window elsewhere.
type TimerWindow struct {
	mu        sync.Mutex
	window    Window
	compact   bool
	saved     Bounds
	maximised bool
}

func NewTimerWindow(window Window) *TimerWindow { return &TimerWindow{window: window} }

func (t *TimerWindow) SetCompact(compact, animate bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.compact == compact {
		return
	}
	w := t.window
	if compact {
		t.maximised = w.IsMaximised()
		if t.maximised {
			w.Unmaximise()
		}
		t.saved = w.Bounds()
		w.SetMinSize(CompactWidth, CompactHeight)
		t.resize(CompactWidth, CompactHeight, animate)
		w.SetMaxSize(CompactWidth, CompactHeight)
		w.SetAlwaysOnTop(true)
	} else {
		w.SetAlwaysOnTop(false)
		w.SetMaxSize(0, 0)
		w.SetPosition(t.saved.X, t.saved.Y)
		t.resize(t.saved.Width, t.saved.Height, animate)
		w.SetMinSize(980, 730)
		if t.maximised {
			w.Maximise()
		}
	}
	t.compact = compact
}

func (t *TimerWindow) resize(width, height int, animate bool) {
	if animate {
		start := t.window.Bounds()
		const duration = 240 * time.Millisecond
		began := time.Now()
		for elapsed := time.Since(began); elapsed < duration; elapsed = time.Since(began) {
			// Cubic ease-out: prompt response, gentle arrival, no overshoot.
			p := 1 - math.Pow(1-float64(elapsed)/float64(duration), 3)
			t.window.SetSize(start.Width+int(math.Round(float64(width-start.Width)*p)), start.Height+int(math.Round(float64(height-start.Height)*p)))
			time.Sleep(15 * time.Millisecond)
		}
	}
	t.window.SetSize(width, height)
}
