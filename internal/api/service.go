package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/bridge"
	"github.com/Karench0/focus-lock/internal/kotlinquiz"
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/profiles"
	"github.com/Karench0/focus-lock/internal/session"
	"github.com/Karench0/focus-lock/internal/statistics"
	"github.com/Karench0/focus-lock/internal/storage"
)

type Service struct {
	mu            sync.Mutex
	store         *storage.Store
	engine        *session.Engine
	bridge        *bridge.Server
	baseline      map[uint32]int64
	failed        map[uint32]int64
	preview       bool
	lastPersist   time.Time
	lastError     string
	notify        func()
	stop          chan struct{}
	done          chan struct{}
	studyPlan     *studyPreparation
	studyQuiz     *kotlinquiz.Quiz
	startingStudy bool
}
type State struct {
	Session    session.Snapshot `json:"session"`
	Settings   model.Settings   `json:"settings"`
	Profiles   []model.Profile  `json:"profiles"`
	Browsers   []bridge.Status  `json:"browsers"`
	Statistics model.Statistics `json:"statistics"`
	Preview    bool             `json:"preview"`
	Error      string           `json:"error"`
}

func New(store *storage.Store, hostPath string, preview bool) (*Service, error) {
	s := &Service{store: store, engine: session.New(time.Now), preview: preview, stop: make(chan struct{}), done: make(chan struct{}), failed: map[uint32]int64{}}
	if !preview {
		b, err := bridge.Start(hostPath, s.rules, s.siteBlocks)
		if err != nil {
			return nil, fmt.Errorf("another instance may be running: %w", err)
		}
		s.bridge = b
	}
	ps, err := store.Profiles()
	if err != nil {
		if s.bridge != nil {
			s.bridge.Close()
		}
		return nil, err
	}
	if len(ps) == 0 {
		for _, name := range []string{"Программирование", "Чтение", "Учёба"} {
			if err = store.SaveProfile(model.Profile{ID: session.ID(), Name: name, Mode: "strict", Applications: []model.Application{}, Websites: []model.Website{}}); err != nil {
				return nil, err
			}
		}
	}
	go s.loop()
	return s, nil
}
func running(s session.Snapshot) bool {
	return s.State == session.Active || s.State == session.ExitPending
}
func (s *Service) immutable() error {
	if running(s.engine.Snapshot()) {
		return errors.New("settings are locked during a session")
	}
	return nil
}
func (s *Service) flush() {
	for _, r := range s.engine.Drain() {
		if err := s.store.Finish(r); err != nil {
			s.lastError = err.Error()
		}
		if s.notify != nil {
			go s.notify()
		}
	}
}
func (s *Service) loop() {
	defer close(s.done)
	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			snap := s.engine.Snapshot()
			s.flush()
			if running(snap) {
				if time.Since(s.lastPersist) >= 3*time.Second {
					if err := s.store.Heartbeat(snap.ID, int(time.Since(snap.StartedAt).Seconds()), snap.AppBlocks, snap.SiteBlocks); err != nil {
						s.lastError = err.Error()
						s.engine.Stop("storage_error")
						s.flush()
					}
					s.lastPersist = time.Now()
				}
				if !s.preview && !s.startingStudy && running(s.engine.Snapshot()) {
					s.monitor(snap)
				}
			}
			s.mu.Unlock()
		}
	}
}
func (s *Service) monitor(snap session.Snapshot) {
	all, err := applications.Snapshot()
	if err != nil {
		s.lastError = err.Error()
		s.engine.Stop("monitor_error")
		s.flush()
		return
	}
	check := func(p applications.Process, ps []applications.Process) bool {
		if snap.Exercise {
			return s.bridge.HealthyStudy(p, ps, snap.ID)
		}
		return s.bridge.Healthy(p, ps, snap.ID)
	}
	for _, p := range all {
		if p.Safe {
			continue
		}
		if !snap.Exercise && s.baseline[p.PID] == p.Created && applications.Browser(p.Path) == "" {
			continue
		}
		if s.failed[p.PID] == p.Created {
			continue
		}
		if applications.Allowed(p, all, snap.Profile.Applications, snap.Profile.Mode == "strict", check) {
			continue
		}
		// A newly-started session gives the extension one lease period to apply
		// its first rules. Start() waits for its acknowledgement before returning.
		if applications.Browser(p.Path) != "" && time.Since(snap.StartedAt) < 6*time.Second {
			continue
		}
		if !running(s.engine.Snapshot()) {
			break
		}
		if err = s.recordTermination(p, applications.Terminate(p)); err != nil {
			s.lastError = fmt.Sprintf("Could not close %s; it may require elevated rights.", p.Name)
			if snap.Exercise {
				s.engine.Stop("monitor_error")
				s.flush()
				break
			}
		}
	}
}

// Both startup and monitoring exempt processes protected by Windows. Remember
// the exact instance so polling does not repeatedly attempt the denied action.
func (s *Service) recordTermination(p applications.Process, err error) error {
	if err != nil {
		s.failed[p.PID] = p.Created
		if applications.IsProtectedProcessError(err) {
			return nil
		}
		return err
	}
	s.engine.AddBlock(false)
	return nil
}

func (s *Service) State() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.engine.Snapshot()
	s.flush()
	settings, err := s.store.Settings()
	if err != nil {
		return State{}, err
	}
	ps, err := s.store.Profiles()
	if err != nil {
		return State{}, err
	}
	records, err := s.store.Records()
	if err != nil {
		return State{}, err
	}
	bs := []bridge.Status{}
	if s.bridge != nil {
		bs = s.bridge.Status()
	}
	return State{Session: snap, Settings: settings, Profiles: ps, Browsers: bs, Statistics: statistics.Aggregate(records, time.Now()), Preview: s.preview, Error: s.lastError}, nil
}
func (s *Service) SaveProfile(p model.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.immutable(); err != nil {
		return err
	}
	if err := profiles.Validate(&p); err != nil {
		return err
	}
	if p.ID == "" {
		p.ID = session.ID()
	}
	for i, a := range p.Applications {
		path, err := applications.Canonical(a.Path)
		if err != nil {
			return err
		}
		p.Applications[i].Path = path
		p.Applications[i].Icon = ""
		if a.Name == "" {
			p.Applications[i].Name = filepath.Base(path)
		}
	}
	return s.store.SaveProfile(p)
}
func (s *Service) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.immutable(); err != nil {
		return err
	}
	ps, err := s.store.Profiles()
	if err != nil {
		return err
	}
	if len(ps) <= 1 {
		return errors.New("keep at least one profile")
	}
	return s.store.DeleteProfile(id)
}
func (s *Service) SaveSettings(v model.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.immutable(); err != nil {
		return err
	}
	if v.Language != "ru" && v.Language != "en" || v.ExitDelaySeconds < 0 || v.ExitDelaySeconds > 3600 {
		return errors.New("invalid settings")
	}
	return s.store.SaveSettings(v)
}
func (s *Service) profile(id string) (model.Profile, error) {
	ps, err := s.store.Profiles()
	if err != nil {
		return model.Profile{}, err
	}
	for _, p := range ps {
		if p.ID == id {
			return p, nil
		}
	}
	return model.Profile{}, errors.New("profile not found")
}
func (s *Service) preflight(p model.Profile) ([]applications.Process, error) {
	if s.preview {
		return []applications.Process{}, nil
	}
	all, err := applications.Snapshot()
	if err != nil {
		return nil, err
	}
	out := []applications.Process{}
	for _, proc := range all {
		if !proc.Visible && applications.Browser(proc.Path) == "" {
			continue
		}
		check := func(pr applications.Process, ps []applications.Process) bool { return s.bridge.Healthy(pr, ps, "") }
		if !applications.Allowed(proc, all, p.Applications, p.Mode == "strict", check) {
			out = append(out, proc)
		}
	}
	return out, nil
}
func (s *Service) Preflight(id string) ([]applications.Process, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.profile(id)
	if err != nil {
		return nil, err
	}
	return s.preflight(p)
}
func (s *Service) Start(id string, minutes int) error {
	s.mu.Lock()
	if err := s.immutable(); err != nil {
		s.mu.Unlock()
		return err
	}
	p, err := s.profile(id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	settings, err := s.store.Settings()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	blocked, err := s.preflight(p)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if len(blocked) > 0 {
		s.mu.Unlock()
		return errors.New("save your work and close the listed applications before starting")
	}
	if !s.preview {
		all, e := applications.Snapshot()
		if e != nil {
			s.mu.Unlock()
			return e
		}
		s.baseline = map[uint32]int64{}
		s.failed = map[uint32]int64{}
		for _, pr := range all {
			s.baseline[pr.PID] = pr.Created
		}
	}
	snap, err := s.engine.Start(p, time.Duration(minutes)*time.Minute, time.Duration(settings.ExitDelaySeconds)*time.Second)
	if err == nil {
		err = s.store.Start(snap.ID, p.Name, snap.StartedAt)
		if err != nil {
			s.engine.Stop("storage_error")
			s.flush()
		}
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if s.preview || p.Mode != "strict" {
		return nil
	}
	// Complete the start call only after each allowed, running browser has
	// actually applied this session's rules. No connection means fail open.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		all, e := applications.Snapshot()
		if e != nil {
			break
		}
		ready := true
		for _, pr := range all {
			if applications.Browser(pr.Path) == "" || pr.Safe {
				continue
			}
			for _, a := range p.Applications {
				if strings.EqualFold(a.Path, pr.Path) && !s.bridge.Healthy(pr, all, snap.ID) {
					ready = false
				}
			}
		}
		if ready {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.mu.Lock()
	s.engine.Stop("extension_error")
	s.flush()
	s.mu.Unlock()
	return errors.New("browser did not acknowledge rules; session cancelled")
}
func (s *Service) RequestExit() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.engine.RequestExit()
	s.flush()
	return err
}
func (s *Service) CancelExit() error   { s.mu.Lock(); defer s.mu.Unlock(); return s.engine.CancelExit() }
func (s *Service) Active() bool        { return running(s.engine.Snapshot()) }
func (s *Service) ClearError()         { s.mu.Lock(); defer s.mu.Unlock(); s.lastError = "" }
func (s *Service) SetNotify(fn func()) { s.mu.Lock(); defer s.mu.Unlock(); s.notify = fn }
func (s *Service) rules() bridge.Rules {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.engine.Snapshot()
	settings, _ := s.store.Settings()
	r := bridge.Rules{Exercise: snap.Exercise, Active: running(snap), Mode: snap.Profile.Mode, Websites: snap.Profile.Websites, Language: settings.Language, LeaseMS: 6000}
	if r.Active {
		r.SessionID = snap.ID
	}
	return r
}
func (s *Service) siteBlocks(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < n; i++ {
		s.engine.AddBlock(true)
	}
}
func (s *Service) Close() {
	close(s.stop)
	<-s.done
	s.mu.Lock()
	s.engine.Stop("shutdown")
	s.flush()
	s.mu.Unlock()
	if s.bridge != nil {
		s.bridge.Close()
	}
	_ = s.store.Close()
}
