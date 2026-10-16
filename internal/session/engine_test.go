package session

import (
	"sync"
	"testing"
	"time"

	"github.com/Karench0/focus-lock/internal/model"
)

func fixture() (*Engine, *time.Time) { n := time.Now(); return New(func() time.Time { return n }), &n }
func start(t *testing.T, e *Engine, d, delay time.Duration) {
	t.Helper()
	if _, err := e.Start(model.Profile{Name: "Work"}, d, delay); err != nil {
		t.Fatal(err)
	}
}
func TestNaturalCompletion(t *testing.T) {
	e, n := fixture()
	start(t, e, time.Minute, time.Second)
	*n = n.Add(time.Minute)
	if got := e.Snapshot(); got.State != Finished || got.Outcome != "completed" {
		t.Fatal(got)
	}
	if len(e.Drain()) != 1 || len(e.Drain()) != 0 {
		t.Fatal("completion must be delivered once")
	}
}
func TestExitCountdownContinuesMainClock(t *testing.T) {
	e, n := fixture()
	start(t, e, 10*time.Minute, time.Minute)
	*n = n.Add(time.Minute)
	if err := e.RequestExit(); err != nil {
		t.Fatal(err)
	}
	*n = n.Add(30 * time.Second)
	s := e.Snapshot()
	if s.State != ExitPending || s.ExitRemainingSeconds != 30 || s.RemainingSeconds != 510 {
		t.Fatal(s)
	}
	if err := e.RequestExit(); err == nil {
		t.Fatal("cannot reset exit countdown")
	}
	*n = n.Add(30 * time.Second)
	if e.Snapshot().Outcome != "interrupted" {
		t.Fatal(e.Snapshot())
	}
}
func TestCancelExitAndRequestAgain(t *testing.T) {
	e, n := fixture()
	start(t, e, 10*time.Minute, 15*time.Second)
	_ = e.RequestExit()
	*n = n.Add(5 * time.Second)
	if err := e.CancelExit(); err != nil {
		t.Fatal(err)
	}
	s := e.Snapshot()
	if s.State != Active || s.ExitEndsAt != nil || s.RemainingSeconds != 595 {
		t.Fatal(s)
	}
	*n = n.Add(20 * time.Second)
	if e.Snapshot().State != Active {
		t.Fatal("cancelled deadline still active")
	}
	_ = e.RequestExit()
	*n = n.Add(15 * time.Second)
	if e.Snapshot().Outcome != "interrupted" {
		t.Fatal(e.Snapshot())
	}
}
func TestEarliestDeadlineWinsWhenSchedulerIsLate(t *testing.T) {
	for _, tt := range []struct {
		delay   time.Duration
		outcome string
	}{{time.Second, "interrupted"}, {time.Minute, "completed"}} {
		e, n := fixture()
		start(t, e, time.Minute, tt.delay)
		*n = n.Add(50 * time.Second)
		_ = e.RequestExit()
		*n = n.Add(time.Minute)
		if e.Snapshot().Outcome != tt.outcome {
			t.Fatalf("want %s: %+v", tt.outcome, e.Snapshot())
		}
	}
}
func TestZeroDelayAndLockedStart(t *testing.T) {
	e, _ := fixture()
	start(t, e, time.Minute, 0)
	if _, err := e.Start(model.Profile{}, time.Minute, 0); err == nil {
		t.Fatal("double start")
	}
	_ = e.RequestExit()
	if e.Snapshot().State != Finished {
		t.Fatal("zero delay should finish immediately")
	}
}
func TestCompletionRaceIsIdempotent(t *testing.T) {
	e, n := fixture()
	start(t, e, time.Minute, time.Minute)
	_ = e.RequestExit()
	*n = n.Add(time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); e.Snapshot(); _ = e.CancelExit(); _ = e.RequestExit(); e.Stop("shutdown") }()
	}
	wg.Wait()
	if len(e.Drain()) != 1 {
		t.Fatal("duplicate finish")
	}
}
func TestShutdownIsFailOpen(t *testing.T) {
	e, _ := fixture()
	start(t, e, time.Minute, 60*time.Second)
	e.Stop("shutdown")
	s := e.Snapshot()
	if s.State != Finished || s.Outcome != "shutdown" {
		t.Fatal(s)
	}
}
func TestRejectInvalidDurations(t *testing.T) {
	e, _ := fixture()
	for _, d := range []time.Duration{0, -time.Minute, 25 * time.Hour} {
		if _, err := e.Start(model.Profile{}, d, time.Second); err == nil {
			t.Fatal(d)
		}
	}
}
