package api

import (
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/storage"
	"path/filepath"
	"testing"
)

func TestBackendLocksMutationsAndCancelsExit(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(db, "", true)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.State()
	if err != nil {
		t.Fatal(err)
	}
	p := state.Profiles[0]
	p.Name = "Renamed"
	if err = s.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	if err = s.Start(p.ID, 1); err != nil {
		t.Fatal(err)
	}
	if s.SaveProfile(p) == nil || s.SaveSettings(model.Settings{Language: "en"}) == nil || s.DeleteProfile(p.ID) == nil {
		t.Fatal("active settings mutation accepted")
	}
	if err = s.RequestExit(); err != nil {
		t.Fatal(err)
	}
	if err = s.CancelExit(); err != nil {
		t.Fatal(err)
	}
	if !s.Active() {
		t.Fatal("cancel ended session")
	}
}
