package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/desktop"
	"github.com/Karench0/focus-lock/internal/kotlinquiz"
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/session"
	"github.com/Karench0/focus-lock/internal/storage"
)

type studyFixture struct {
	service *Service
	before  State
	path    string
	close   func()
}

func newStudyFixture(t *testing.T) studyFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "study.db")
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Save deliberate distractions directly to this disposable database. No
	// real executable, browser, process, or user's settings are touched.
	profile := model.Profile{
		ID: "personal-profile", Name: "Personal preferences", Mode: "blocklist",
		Applications: []model.Application{{Name: "Steam", Path: `C:\Games\Steam\steam.exe`}},
		Websites:     []model.Website{{Domain: "youtube.com", Subdomains: true}, {Domain: "tiktok.com"}},
	}
	if err := db.SaveProfile(profile); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.SaveSettings(model.Settings{Language: "en", ExitDelaySeconds: 17, Onboarded: true}); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	s, err := New(db, "", true)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	closeService := sync.OnceFunc(s.Close)
	t.Cleanup(closeService)
	before, err := s.State()
	if err != nil {
		t.Fatal(err)
	}
	return studyFixture{service: s, before: before, path: path, close: closeService}
}

func studyCorrectIndex(t *testing.T, question kotlinquiz.Question) int {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "..", "tests", "oracles", "kotlin-quiz.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle map[string]string
	if err := json.Unmarshal(encoded, &oracle); err != nil {
		t.Fatal(err)
	}
	index := slices.Index(question.Options, oracle[question.Prompt])
	if index < 0 {
		t.Fatalf("question not present in source-only integration oracle: %q", question.Prompt)
	}
	return index
}

func TestStorozhenkoPreparationDoesNotStartOrPersistMode(t *testing.T) {
	f := newStudyFixture(t)
	s := f.service
	if s.Active() || s.rules().Active || f.before.Session.Exercise {
		t.Fatal("exercise started automatically")
	}
	if _, err := s.KotlinQuestion(); err == nil {
		t.Fatal("question available outside an exercise")
	}
	if correct, err := s.AnswerKotlin("missing", 0); err == nil || correct {
		t.Fatal("answer accepted outside an exercise")
	}
	plan, err := s.PrepareStorozhenko()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Token == "" || !plan.BrowserReady || plan.DocsURL != desktop.KotlinDocsURL {
		t.Fatalf("invalid preview plan: %#v", plan)
	}
	if s.Active() || s.rules().Active {
		t.Fatal("preparation enabled enforcement before consent")
	}
	state, err := s.State()
	if err != nil || !reflect.DeepEqual(state.Settings, f.before.Settings) || !reflect.DeepEqual(state.Profiles, f.before.Profiles) {
		t.Fatalf("preparation rewrote user preferences: %v", err)
	}
	newPlan, err := s.PrepareStorozhenko()
	if err != nil || newPlan.Token == plan.Token {
		t.Fatal("new preparation did not replace its consent token")
	}
	if err := s.StartStorozhenko(plan.Token, "storozhenko", true); err == nil {
		t.Fatal("superseded preparation token was accepted")
	}
}

func TestStorozhenkoStartRequiresCurrentExplicitConsent(t *testing.T) {
	for _, name := range []string{"no preparation", "empty token", "wrong token", "wrong person", "empty person", "not confirmed", "expired", "browser not ready"} {
		t.Run(name, func(t *testing.T) {
			f := newStudyFixture(t)
			s := f.service
			plan, err := s.PrepareStorozhenko()
			if err != nil {
				t.Fatal(err)
			}
			token, person, confirmed := plan.Token, "storozhenko", true
			s.mu.Lock()
			switch name {
			case "no preparation":
				s.studyPlan = nil
			case "empty token":
				token = ""
			case "wrong token":
				token += "x"
			case "wrong person":
				person = "someone else"
			case "empty person":
				person = ""
			case "not confirmed":
				confirmed = false
			case "expired":
				s.studyPlan.at = time.Now().Add(-3 * time.Minute)
			case "browser not ready":
				s.studyPlan.view.BrowserReady = false
			}
			s.mu.Unlock()
			if err := s.StartStorozhenko(token, person, confirmed); err == nil {
				t.Fatal("invalid consent started exercise")
			}
			if s.Active() || s.rules().Active {
				t.Fatal("rejected consent left enforcement active")
			}
		})
	}
}

func TestStorozhenkoQuestionIsOnlyExitAndLeavesPreferencesIntact(t *testing.T) {
	f := newStudyFixture(t)
	s := f.service
	plan, err := s.PrepareStorozhenko()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.StartStorozhenko(plan.Token, "storozhenko", true); err != nil {
		t.Fatal(err)
	}
	state, err := s.State()
	if err != nil {
		t.Fatal(err)
	}
	id := state.Session.ID
	if id == "" || !state.Session.Exercise || state.Session.State != session.Active || !state.Session.EndsAt.IsZero() {
		t.Fatalf("exercise did not start as untimed active session: %#v", state.Session)
	}
	rules := s.rules()
	if !rules.Active || rules.SessionID != id || rules.Mode != "strict" || len(rules.Websites) != 1 || rules.Websites[0].Domain != "kotlinlang.org" {
		t.Fatalf("personal distraction preferences leaked into temporary rules: %#v", rules)
	}
	if err := s.RequestExit(); err == nil {
		t.Fatal("ordinary early exit bypassed the quiz")
	}
	if err := s.CancelExit(); err == nil {
		t.Fatal("exercise unexpectedly entered the ordinary exit state")
	}
	if s.SaveSettings(f.before.Settings) == nil || s.SaveProfile(f.before.Profiles[0]) == nil || s.DeleteProfile(f.before.Profiles[0].ID) == nil {
		t.Fatal("active exercise accepted saved-preference mutations")
	}
	if _, err := s.PrepareStorozhenko(); err == nil {
		t.Fatal("active exercise accepted a second preparation")
	}
	if err := s.StartStorozhenko(plan.Token, "storozhenko", true); err == nil {
		t.Fatal("consent token could start a second active exercise")
	}
	question, err := s.KotlinQuestion()
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.KotlinQuestion()
	if err != nil || !reflect.DeepEqual(question, again) {
		t.Fatal("reopening the quiz rerolled a pending challenge")
	}
	encoded, err := json.Marshal(question)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil || len(fields) != 4 {
		t.Fatalf("question leaked private fields: %s (%v)", encoded, err)
	}
	for _, option := range []int{-1, 8} {
		if correct, err := s.AnswerKotlin(question.Token, option); err == nil || correct {
			t.Fatal("invalid option accepted")
		}
	}
	if correct, err := s.AnswerKotlin("foreign-token", 0); err == nil || correct {
		t.Fatal("foreign token accepted")
	}
	again, err = s.KotlinQuestion()
	if err != nil || !reflect.DeepEqual(question, again) {
		t.Fatal("invalid answer consumed pending question")
	}
	wrong := (studyCorrectIndex(t, question) + 1) % len(question.Options)
	if correct, err := s.AnswerKotlin(question.Token, wrong); err != nil || correct {
		t.Fatalf("wrong answer result: %v %v", correct, err)
	}
	state, err = s.State()
	if err != nil || !s.Active() || state.Session.ID != id || !s.rules().Active {
		t.Fatal("wrong answer ended or replaced the exercise")
	}
	next, err := s.KotlinQuestion()
	if err != nil || next.Token == question.Token || next.Prompt == question.Prompt {
		t.Fatal("wrong answer did not produce a different one-use challenge")
	}
	if correct, err := s.AnswerKotlin(question.Token, studyCorrectIndex(t, question)); err == nil || correct {
		t.Fatal("replaying previous token could finish the exercise")
	}
	again, err = s.KotlinQuestion()
	if err != nil || !reflect.DeepEqual(next, again) {
		t.Fatal("old-token replay consumed the next challenge")
	}
	if correct, err := s.AnswerKotlin(next.Token, studyCorrectIndex(t, next)); err != nil || !correct {
		t.Fatalf("correct answer rejected: %v %v", correct, err)
	}
	state, err = s.State()
	if err != nil || s.Active() || state.Session.ID != id || state.Session.State != session.Finished || state.Session.Outcome != "completed" {
		t.Fatalf("correct answer did not complete original exercise: %#v (%v)", state.Session, err)
	}
	if s.rules().Active || s.rules().SessionID != "" {
		t.Fatal("exercise policy remained active after correct answer")
	}
	if _, err := s.KotlinQuestion(); err == nil {
		t.Fatal("question remained available after completion")
	}
	if correct, err := s.AnswerKotlin(next.Token, studyCorrectIndex(t, next)); err == nil || correct {
		t.Fatal("completion token could be replayed")
	}
	if !reflect.DeepEqual(state.Settings, f.before.Settings) || !reflect.DeepEqual(state.Profiles, f.before.Profiles) {
		t.Fatal("temporary exercise rewrote saved user preferences")
	}
	if err := s.Start(f.before.Profiles[0].ID, 1); err != nil {
		t.Fatal(err)
	}
	normal, err := s.State()
	if err != nil || normal.Session.Exercise || normal.Session.ID == id || normal.Session.Profile.ID != f.before.Profiles[0].ID {
		t.Fatal("normal session retained the exercise policy")
	}
	if err := s.RequestExit(); err != nil {
		t.Fatalf("normal session lost its standard exit behavior: %v", err)
	}
}

func TestStorozhenkoQuestionsRejectedWhileStartPending(t *testing.T) {
	f := newStudyFixture(t)
	s := f.service
	plan, err := s.PrepareStorozhenko()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.StartStorozhenko(plan.Token, "storozhenko", true); err != nil {
		t.Fatal(err)
	}
	question, err := s.KotlinQuestion()
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.startingStudy = true
	s.mu.Unlock()
	if _, err := s.KotlinQuestion(); err == nil {
		t.Fatal("quiz available before start finished")
	}
	if correct, err := s.AnswerKotlin(question.Token, studyCorrectIndex(t, question)); err == nil || correct {
		t.Fatal("exercise completed while start was still applying enforcement")
	}
	s.mu.Lock()
	s.startingStudy = false
	s.mu.Unlock()
	again, err := s.KotlinQuestion()
	if err != nil || !reflect.DeepEqual(again, question) {
		t.Fatal("rejected answer consumed pending challenge")
	}
}

func TestStorozhenkoNeverRestartsAutomatically(t *testing.T) {
	f := newStudyFixture(t)
	plan, err := f.service.PrepareStorozhenko()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.StartStorozhenko(plan.Token, "storozhenko", true); err != nil {
		t.Fatal(err)
	}
	f.close()
	db, err := storage.Open(f.path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(db, "", true)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	defer reopened.Close()
	state, err := reopened.State()
	if err != nil || reopened.Active() || state.Session.Exercise || reopened.rules().Active {
		t.Fatalf("exercise reactivated after restart: %#v %v", state.Session, err)
	}
	if !reflect.DeepEqual(state.Settings, f.before.Settings) || !reflect.DeepEqual(state.Profiles, f.before.Profiles) {
		t.Fatal("exercise persisted a modified user profile or hidden enabled setting")
	}
	if err := reopened.StartStorozhenko(plan.Token, "storozhenko", true); err == nil {
		t.Fatal("old-process consent token survived restart")
	}
}

func TestStudyProfileIsStrictAndOnlyAllowsKotlinTools(t *testing.T) {
	browser := model.Application{Name: "Browser", Path: `C:\Browser\chrome.exe`}
	installed := []model.Application{
		{Name: "IntelliJ IDEA", Path: `C:\IDE\bin\idea64.exe`, Icon: "large-icon"},
		{Name: "Android Studio", Path: `C:\Studio\bin\studio64.exe`, Icon: "large-icon"},
		{Name: "Steam", Path: `C:\Steam\steam.exe`, AllowChildren: true},
		{Name: "Dota 2", Path: `C:\Steam\dota2.exe`},
		{Name: "Editor", Path: `C:\Editor\code.exe`},
		{Name: "Shell", Path: `C:\Tools\powershell.exe`},
	}
	profile := studyProfile(browser, installed)
	if profile.Mode != "strict" || len(profile.Websites) != 1 || profile.Websites[0].Domain != "kotlinlang.org" || profile.Websites[0].Subdomains {
		t.Fatalf("unexpected exercise websites: %#v", profile)
	}
	if !reflect.DeepEqual(profile.Websites[0].Resources, []string{"resources.jetbrains.com"}) {
		t.Fatalf("unexpected resource permissions: %#v", profile.Websites[0].Resources)
	}
	if len(profile.Applications) != 3 || profile.Applications[0] != browser {
		t.Fatalf("unexpected allowed applications: %#v", profile.Applications)
	}
	for _, ide := range profile.Applications[1:] {
		if !ide.AllowChildren || ide.Icon != "" {
			t.Fatalf("IDE permissions were not limited to local install children: %#v", ide)
		}
	}
	check := func(applications.Process, []applications.Process) bool { return true }
	for _, distraction := range []applications.Process{
		{Path: `C:\Steam\steam.exe`}, {Path: `C:\Steam\dota2.exe`},
		{Path: `C:\Editor\code.exe`}, {Path: `C:\Tools\powershell.exe`},
		{Path: `C:\Other\java.exe`},
	} {
		if applications.Allowed(distraction, nil, profile.Applications, true, check) {
			t.Errorf("unrelated application allowed: %s", distraction.Path)
		}
	}
	if installed[0].Icon != "large-icon" || installed[0].AllowChildren {
		t.Fatal("building a temporary profile mutated discovered applications")
	}
}
