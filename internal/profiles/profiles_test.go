package profiles

import (
	"github.com/Karench0/focus-lock/internal/model"
	"testing"
)

func TestProfileValidation(t *testing.T) {
	p := model.Profile{Name: " Work ", Mode: "strict", Websites: []model.Website{{Domain: "GO.DEV"}}}
	if err := Validate(&p); err != nil || p.Name != "Work" || p.Websites[0].Domain != "go.dev" {
		t.Fatal(p, err)
	}
	p.Websites = append(p.Websites, model.Website{Domain: "go.dev"})
	if Validate(&p) == nil {
		t.Fatal("duplicate domain accepted")
	}
}
