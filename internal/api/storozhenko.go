package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/desktop"
	"github.com/Karench0/focus-lock/internal/kotlinquiz"
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/session"
)

type StudyPlan struct {
	Token        string                 `json:"token"`
	Blocked      []applications.Process `json:"blocked"`
	BrowserReady bool                   `json:"browserReady"`
	DocsURL      string                 `json:"docsUrl"`
}
type studyPreparation struct {
	view    StudyPlan
	profile model.Profile
	at      time.Time
}

// The exercise is a temporary policy. It never rewrites the user's profiles or
// persists a hidden enabled flag which could surprise a different user later.
func studyProfile(browser model.Application, installed []model.Application) model.Profile {
	allowed := []model.Application{browser}
	for _, app := range installed {
		switch strings.ToLower(filepath.Base(app.Path)) {
		case "idea.exe", "idea64.exe", "studio.exe", "studio64.exe":
			app.Icon = ""
			app.AllowChildren = true
			allowed = append(allowed, app)
			// JetBrains keeps its bundled runtime next to bin, outside the
			// executable directory covered by the normal child-process rule.
			// Permit only verified helper paths from this IDE's own runtime.
			for _, name := range []string{"java.exe", "javaw.exe", "jcef_helper.exe"} {
				path, err := applications.Canonical(filepath.Join(filepath.Dir(filepath.Dir(app.Path)), "jbr", "bin", name))
				if err == nil {
					allowed = append(allowed, model.Application{Name: app.Name + " · " + name, Path: path, AllowChildren: true})
				}
			}
		}
	}
	return model.Profile{ID: "storozhenko-kotlin", Name: "Storozhenko · Kotlin", Mode: "strict", Applications: allowed, Websites: []model.Website{{Domain: "kotlinlang.org", Resources: []string{"resources.jetbrains.com"}}}}
}

func (s *Service) studyBlocked(p model.Profile, all []applications.Process, id string) []applications.Process {
	out := []applications.Process{}
	for _, pr := range all {
		check := func(proc applications.Process, ps []applications.Process) bool {
			return s.bridge.HealthyStudy(proc, ps, id)
		}
		if !applications.Allowed(pr, all, p.Applications, true, check) {
			out = append(out, pr)
		}
	}
	return out
}

func (s *Service) PrepareStorozhenko() (StudyPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.immutable(); err != nil {
		return StudyPlan{}, err
	}
	view := StudyPlan{Token: session.ID(), Blocked: []applications.Process{}, DocsURL: desktop.KotlinDocsURL}
	var profile model.Profile
	if s.preview {
		view.BrowserReady = true
		profile = studyProfile(model.Application{Name: "Preview browser", Path: `C:\Preview\chrome.exe`}, nil)
	} else {
		browser, err := applications.DefaultBrowser()
		if err != nil {
			return StudyPlan{}, err
		}
		profile = studyProfile(browser, applications.Discover())
		all, err := applications.Snapshot()
		if err != nil {
			return StudyPlan{}, err
		}
		for _, pr := range all {
			if strings.EqualFold(pr.Path, browser.Path) && s.bridge.HealthyStudy(pr, all, "") {
				view.BrowserReady = true
			}
		}
		view.Blocked = s.studyBlocked(profile, all, "")
	}
	s.studyPlan = &studyPreparation{view: view, profile: profile, at: time.Now()}
	return view, nil
}

func (s *Service) StartStorozhenko(token, self string, confirmed bool) error {
	s.mu.Lock()
	if err := s.immutable(); err != nil {
		s.mu.Unlock()
		return err
	}
	plan := s.studyPlan
	if !confirmed || self != "storozhenko" || plan == nil || plan.view.Token != token || time.Since(plan.at) > 2*time.Minute || !plan.view.BrowserReady {
		s.mu.Unlock()
		return errors.New("prepare and explicitly confirm the Storozhenko exercise first")
	}
	if !s.preview {
		browser, err := applications.DefaultBrowser()
		if err != nil || !strings.EqualFold(browser.Path, plan.profile.Applications[0].Path) {
			s.mu.Unlock()
			return errors.New("default browser changed; review the exercise again")
		}
	}
	snap, err := s.engine.StartExercise(plan.profile)
	if err == nil {
		err = s.store.Start(snap.ID, plan.profile.Name, snap.StartedAt)
	}
	if err != nil {
		s.engine.Stop("storage_error")
		s.flush()
		s.mu.Unlock()
		return err
	}
	s.studyPlan = nil
	s.studyQuiz = kotlinquiz.New()
	s.startingStudy = true
	s.failed = map[uint32]int64{}
	s.mu.Unlock()

	ready := s.preview
	if !ready {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			for _, status := range s.bridge.Status() {
				if status.Connected && status.StudyCapable && status.Applied == snap.ID && strings.EqualFold(status.Path, plan.profile.Applications[0].Path) {
					ready = true
				}
			}
			if ready {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() { s.startingStudy = false }()
	fail := func(err error) error {
		s.engine.Stop("exercise_start_error")
		s.flush()
		s.studyQuiz.Reset()
		return err
	}
	if current := s.engine.Snapshot(); current.ID != snap.ID || !current.Exercise || !running(current) {
		return errors.New("exercise start was cancelled")
	}
	if !ready {
		return fail(errors.New("default browser did not apply the rules; exercise cancelled"))
	}
	if !s.preview {
		all, err := applications.Snapshot()
		if err != nil {
			return fail(err)
		}
		blocked := s.studyBlocked(plan.profile, all, snap.ID)
		approved := map[uint32]int64{}
		for _, pr := range plan.view.Blocked {
			approved[pr.PID] = pr.Created
		}
		for _, pr := range blocked {
			if approved[pr.PID] != pr.Created {
				return fail(errors.New("open applications changed; review the exercise again"))
			}
		}
		if err = desktop.OpenKotlinDocs(); err != nil {
			return fail(err)
		}
		for _, pr := range blocked {
			if err = applications.Terminate(pr); err != nil {
				return fail(fmt.Errorf("could not close %s; exercise cancelled: %w", pr.Name, err))
			}
			s.engine.AddBlock(false)
		}
	}
	return nil
}

func (s *Service) KotlinQuestion() (kotlinquiz.Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.engine.Snapshot()
	if s.startingStudy || !snap.Exercise || !running(snap) || s.studyQuiz == nil {
		return kotlinquiz.Question{}, errors.New("no active Storozhenko exercise")
	}
	return s.studyQuiz.Get(snap.ID)
}

func (s *Service) AnswerKotlin(token string, option int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.engine.Snapshot()
	if s.startingStudy || !snap.Exercise || !running(snap) || s.studyQuiz == nil {
		return false, errors.New("no active Storozhenko exercise")
	}
	correct, err := s.studyQuiz.Answer(snap.ID, token, option)
	if err != nil || !correct {
		return false, err
	}
	if err = s.engine.CompleteExercise(); err != nil {
		return false, err
	}
	s.flush()
	s.studyQuiz.Reset()
	return true, nil
}
