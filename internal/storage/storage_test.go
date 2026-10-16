package storage

import (
	"github.com/Karench0/focus-lock/internal/model"
	"path/filepath"
	"testing"
	"time"
)

func TestProfilesSettingsAndCrashRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := s.Settings()
	if settings.Language != "ru" || settings.ExitDelaySeconds != 60 {
		t.Fatal(settings)
	}
	p := model.Profile{ID: "one", Name: "Work", Mode: "strict"}
	if err = s.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	p.Name = "Reading"
	_ = s.SaveProfile(p)
	ps, err := s.Profiles()
	if err != nil || len(ps) != 1 || ps[0].Name != "Reading" {
		t.Fatal(ps, err)
	}
	_ = s.Start("session", "Work", time.Now())
	_ = s.Heartbeat("session", 5, 1, 2)
	_ = s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	records, err := s.Records()
	if err != nil || len(records) != 1 || records[0].Outcome != "recovered" || records[0].Seconds != 5 {
		t.Fatal(records, err)
	}
	_ = s.DeleteProfile("one")
	ps, _ = s.Profiles()
	if len(ps) != 0 {
		t.Fatal(ps)
	}
}
func TestFinishIsIdempotent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Start("s", "Work", time.Now())
	r := model.Record{ID: "s", Outcome: "completed", Seconds: 60, EndedAt: time.Now().Format(time.RFC3339)}
	if err = s.Finish(r); err != nil {
		t.Fatal(err)
	}
	r.Seconds = 100
	_ = s.Finish(r)
	rs, _ := s.Records()
	if len(rs) != 1 || rs[0].Seconds != 60 {
		t.Fatal(rs)
	}
}
