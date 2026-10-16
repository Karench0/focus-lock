package desktop

import "testing"

type testWindow struct {
	b                      Bounds
	minW, minH, maxW, maxH int
	top, maximised         bool
	frames                 []Bounds
}

func (w *testWindow) Bounds() Bounds { return w.b }
func (w *testWindow) SetSize(x, y int) {
	w.b.Width = x
	w.b.Height = y
	w.frames = append(w.frames, w.b)
}
func (w *testWindow) SetPosition(x, y int)   { w.b.X = x; w.b.Y = y }
func (w *testWindow) SetMinSize(x, y int)    { w.minW = x; w.minH = y }
func (w *testWindow) SetMaxSize(x, y int)    { w.maxW = x; w.maxH = y }
func (w *testWindow) SetAlwaysOnTop(on bool) { w.top = on }
func (w *testWindow) IsMaximised() bool      { return w.maximised }
func (w *testWindow) Unmaximise()            { w.maximised = false }
func (w *testWindow) Maximise()              { w.maximised = true }

func TestCompactDragAndRestore(t *testing.T) {
	w := &testWindow{b: Bounds{-1100, 40, 1220, 850}}
	saved := w.b
	timer := NewTimerWindow(w)
	timer.SetCompact(true, false)
	if w.b.Width != CompactWidth || w.b.Height != CompactHeight || !w.top || w.maxW != CompactWidth || w.minH != CompactHeight {
		t.Fatal("invalid compact bounds", w)
	}
	w.SetPosition(120, 400)
	timer.SetCompact(true, false) // Duplicate requests must not overwrite saved bounds.
	timer.SetCompact(false, false)
	if w.b != saved || w.top || w.maxW != 0 || w.minW != 980 || w.minH != 730 {
		t.Fatal("restore lost original geometry", w)
	}
	timer.SetCompact(false, false)
	if w.b != saved {
		t.Fatal("duplicate restore")
	}
}
func TestMaximisedRestoreAndAnimation(t *testing.T) {
	w := &testWindow{b: Bounds{20, 30, 1220, 850}, maximised: true}
	timer := NewTimerWindow(w)
	timer.SetCompact(true, true)
	if w.maximised || len(w.frames) < 2 {
		t.Fatal("window did not animate from maximised state")
	}
	for i := 1; i < len(w.frames); i++ {
		if w.frames[i].Width > w.frames[i-1].Width || w.frames[i].Height > w.frames[i-1].Height {
			t.Fatal("animation overshot")
		}
	}
	timer.SetCompact(false, false)
	if !w.maximised || w.b.Width != 1220 {
		t.Fatal("maximised state lost")
	}
}
