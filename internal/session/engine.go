// Package session owns the authoritative, monotonic session clock.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/Karench0/focus-lock/internal/model"
)

type State string

const (
	Idle        State = "IDLE"
	Active      State = "ACTIVE"
	ExitPending State = "EXIT_PENDING"
	Finished    State = "FINISHED"
)

type Snapshot struct {
	Exercise             bool          `json:"exercise"`
	ElapsedSeconds       int           `json:"elapsedSeconds"`
	State                State         `json:"state"`
	ID                   string        `json:"id"`
	Profile              model.Profile `json:"profile"`
	StartedAt            time.Time     `json:"startedAt"`
	EndsAt               time.Time     `json:"endsAt"`
	ExitStartedAt        *time.Time    `json:"exitStartedAt"`
	ExitEndsAt           *time.Time    `json:"exitEndsAt"`
	RemainingSeconds     int           `json:"remainingSeconds"`
	ExitRemainingSeconds int           `json:"exitRemainingSeconds"`
	DurationSeconds      int           `json:"durationSeconds"`
	Progress             float64       `json:"progress"`
	Outcome              string        `json:"outcome"`
	AppBlocks            int           `json:"appBlocks"`
	SiteBlocks           int           `json:"siteBlocks"`
}

type Engine struct {
	mu       sync.Mutex
	now      func() time.Time
	s        Snapshot
	delay    time.Duration
	finished []model.Record
}

func New(now func() time.Time) *Engine {
	return &Engine{now: now, s: Snapshot{State: Idle}}
}

func ID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (e *Engine) running() bool { return e.s.State == Active || e.s.State == ExitPending }
func (e *Engine) Start(p model.Profile, duration, delay time.Duration) (Snapshot, error) {
	return e.start(p, duration, delay, false)
}

// Study exercises end on a validated answer, not a timer or an early-exit delay.
func (e *Engine) StartExercise(p model.Profile) (Snapshot, error) {
	return e.start(p, 0, 0, true)
}

func (e *Engine) start(p model.Profile, duration, delay time.Duration, exercise bool) (Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tick()
	if e.running() {
		return e.s, errors.New("session already active")
	}
	if !exercise && (duration < time.Minute || duration > 24*time.Hour || delay < 0 || delay > time.Hour) {
		return e.s, errors.New("invalid duration or exit delay")
	}
	n := e.now()
	e.s = Snapshot{State: Active, ID: ID(), Profile: p, StartedAt: n, EndsAt: n.Add(duration), DurationSeconds: int(duration.Seconds())}
	e.s.Exercise = exercise
	if exercise {
		e.s.EndsAt = time.Time{}
	}
	e.delay = delay
	e.tick()
	return e.s, nil
}

func (e *Engine) RequestExit() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tick()
	if e.s.Exercise {
		return errors.New("answer a Kotlin question to finish this exercise")
	}
	if e.s.State != Active {
		return errors.New("no active session")
	}
	n := e.now()
	end := n.Add(e.delay)
	e.s.State = ExitPending
	e.s.ExitStartedAt = &n
	e.s.ExitEndsAt = &end
	e.tick()
	return nil
}

func (e *Engine) CompleteExercise() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.s.Exercise || e.s.State != Active {
		return errors.New("no active exercise")
	}
	e.finish(e.now(), "completed")
	return nil
}
func (e *Engine) CancelExit() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tick()
	if e.s.State != ExitPending {
		return errors.New("exit is not pending")
	}
	e.s.State = Active
	e.s.ExitStartedAt = nil
	e.s.ExitEndsAt = nil
	e.s.ExitRemainingSeconds = 0
	return nil
}
func (e *Engine) Snapshot() Snapshot { e.mu.Lock(); defer e.mu.Unlock(); e.tick(); return e.s }
func (e *Engine) AddBlock(site bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tick()
	if !e.running() {
		return
	}
	if site {
		e.s.SiteBlocks++
	} else {
		e.s.AppBlocks++
	}
}
func (e *Engine) Stop(outcome string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tick()
	if e.running() {
		e.finish(e.now(), outcome)
	}
}
func (e *Engine) Drain() []model.Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.finished
	e.finished = nil
	return r
}
func (e *Engine) tick() {
	if !e.running() {
		return
	}
	n := e.now()
	e.s.ElapsedSeconds = max(0, int(n.Sub(e.s.StartedAt).Seconds()))
	if e.s.Exercise {
		return
	}
	// Select the earliest deadline even when both expire between scheduler ticks.
	deadline := e.s.EndsAt
	outcome := "completed"
	if e.s.State == ExitPending && e.s.ExitEndsAt.Before(deadline) {
		deadline = *e.s.ExitEndsAt
		outcome = "interrupted"
	}
	if !n.Before(deadline) {
		e.finish(deadline, outcome)
		return
	}
	e.s.RemainingSeconds = int(math.Ceil(e.s.EndsAt.Sub(n).Seconds()))
	e.s.Progress = math.Max(0, math.Min(1, n.Sub(e.s.StartedAt).Seconds()/float64(e.s.DurationSeconds)))
	if e.s.ExitEndsAt != nil {
		e.s.ExitRemainingSeconds = int(math.Ceil(e.s.ExitEndsAt.Sub(n).Seconds()))
	}
}
func (e *Engine) finish(at time.Time, outcome string) {
	e.s.State = Finished
	e.s.Outcome = outcome
	e.s.ElapsedSeconds = max(0, int(at.Sub(e.s.StartedAt).Seconds()))
	if e.s.DurationSeconds > 0 {
		e.s.Progress = math.Max(0, math.Min(1, at.Sub(e.s.StartedAt).Seconds()/float64(e.s.DurationSeconds)))
	}
	e.s.RemainingSeconds = 0
	e.s.ExitRemainingSeconds = 0
	e.finished = append(e.finished, model.Record{ID: e.s.ID, Profile: e.s.Profile.Name, StartedAt: e.s.StartedAt.Format(time.RFC3339), EndedAt: at.Format(time.RFC3339), Seconds: int(at.Sub(e.s.StartedAt).Seconds()), Outcome: outcome, AppBlocks: e.s.AppBlocks, SiteBlocks: e.s.SiteBlocks})
}
