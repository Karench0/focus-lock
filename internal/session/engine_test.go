package session

import (
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
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

func TestExerciseRemainsUntimedUntilExplicitCompletion(t *testing.T) {
	e, n := fixture()
	profile := model.Profile{ID: "exercise", Name: "Kotlin", Mode: "strict"}
	initial, err := e.StartExercise(profile)
	if err != nil {
		t.Fatal(err)
	}
	if initial.State != Active || !initial.Exercise || !initial.EndsAt.IsZero() || initial.DurationSeconds != 0 {
		t.Fatalf("invalid untimed exercise: %#v", initial)
	}
	// Exercise uses a zero exit delay internally. Ordinary exit must still be
	// rejected before any timer logic can complete it immediately.
	if err := e.RequestExit(); err == nil {
		t.Fatal("zero-delay exit bypassed exercise completion")
	}
	elapsed := 49*time.Hour + 17*time.Second
	*n = n.Add(elapsed)
	active := e.Snapshot()
	if active.State != Active || !active.Exercise || active.ID != initial.ID || active.ElapsedSeconds != int(elapsed.Seconds()) {
		t.Fatalf("exercise expired or lost elapsed time after 48 hours: %#v", active)
	}
	if active.RemainingSeconds != 0 || active.ExitRemainingSeconds != 0 || active.ExitStartedAt != nil || active.ExitEndsAt != nil {
		t.Fatalf("untimed exercise created timer deadlines: %#v", active)
	}
	if math.IsInf(active.Progress, 0) || math.IsNaN(active.Progress) {
		t.Fatalf("untimed progress is not finite: %v", active.Progress)
	}
	if _, err := json.Marshal(active); err != nil {
		t.Fatalf("active untimed snapshot cannot be sent as JSON: %v", err)
	}
	if err := e.RequestExit(); err == nil || e.Snapshot().State != Active {
		t.Fatal("late ordinary exit completed the exercise")
	}
	if records := e.Drain(); len(records) != 0 {
		t.Fatalf("untimed exercise emitted a record before completion: %#v", records)
	}
	if err := e.CompleteExercise(); err != nil {
		t.Fatal(err)
	}
	if err := e.CompleteExercise(); err == nil {
		t.Fatal("exercise could be completed twice")
	}
	finished := e.Snapshot()
	if finished.State != Finished || finished.Outcome != "completed" || finished.ElapsedSeconds != int(elapsed.Seconds()) {
		t.Fatalf("wrong exercise completion: %#v", finished)
	}
	if math.IsInf(finished.Progress, 0) || math.IsNaN(finished.Progress) {
		t.Fatalf("finished untimed progress is not finite: %v", finished.Progress)
	}
	if _, err := json.Marshal(finished); err != nil {
		t.Fatalf("finished untimed snapshot cannot be sent as JSON: %v", err)
	}
	records := e.Drain()
	if len(records) != 1 || records[0].ID != initial.ID || records[0].Profile != profile.Name || records[0].Outcome != "completed" || records[0].Seconds != int(elapsed.Seconds()) {
		t.Fatalf("completion must emit the original exercise once: %#v", records)
	}
	if len(e.Drain()) != 0 {
		t.Fatal("exercise record delivered twice")
	}
	// Starting an ordinary session resets the untimed policy and preserves its
	// existing zero-delay early-exit behavior.
	start(t, e, time.Minute, 0)
	if e.Snapshot().Exercise {
		t.Fatal("ordinary session inherited exercise mode")
	}
	if err := e.CompleteExercise(); err == nil {
		t.Fatal("quiz completion could finish an ordinary session")
	}
	if err := e.RequestExit(); err != nil || e.Snapshot().State != Finished {
		t.Fatalf("ordinary zero-delay exit regressed: %v", err)
	}
}

func TestExerciseConcurrentCompletionRecordsOnce(t *testing.T) {
	e, n := fixture()
	if _, err := e.StartExercise(model.Profile{Name: "Kotlin"}); err != nil {
		t.Fatal(err)
	}
	*n = n.Add(50 * time.Hour)
	var successful atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			e.Snapshot()
			if e.CompleteExercise() == nil {
				successful.Add(1)
			}
		})
	}
	wg.Wait()
	if successful.Load() != 1 {
		t.Fatalf("concurrent completions accepted %d times", successful.Load())
	}
	records := e.Drain()
	if len(records) != 1 || records[0].Outcome != "completed" || records[0].Seconds != int((50*time.Hour).Seconds()) {
		t.Fatalf("concurrent completions did not produce one complete record: %#v", records)
	}
}
