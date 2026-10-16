package profiles

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/websites"
)

func Validate(p *model.Profile) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len([]rune(p.Name)) > 80 {
		return errors.New("profile name must contain 1–80 characters")
	}
	if p.Mode != "strict" && p.Mode != "blocklist" {
		return errors.New("invalid website mode")
	}
	if len(p.Websites) > 200 || len(p.Applications) > 200 {
		return errors.New("at most 200 rules of each kind")
	}
	seen := map[string]bool{}
	resourceCount := 0
	for i, w := range p.Websites {
		d, err := websites.Normalize(w.Domain)
		if err != nil {
			return err
		}
		if seen[d] {
			return errors.New("duplicate domain")
		}
		seen[d] = true
		p.Websites[i].Domain = d
		if len(w.Resources) > 16 {
			return errors.New("at most 16 resource domains per website")
		}
		resourceSeen := map[string]bool{}
		for j, raw := range w.Resources {
			resource, err := websites.Normalize(raw)
			if err != nil {
				return err
			}
			if resourceSeen[resource] {
				return errors.New("duplicate resource domain")
			}
			resourceSeen[resource] = true
			p.Websites[i].Resources[j] = resource
			resourceCount++
		}
	}
	if resourceCount > 200 {
		return errors.New("at most 200 resource domains per profile")
	}
	encoded, err := json.Marshal(p.Websites)
	if err != nil {
		return err
	}
	if len(encoded) > 60000 {
		return errors.New("website policy exceeds the native messaging size limit")
	}
	if p.Applications == nil {
		p.Applications = []model.Application{}
	}
	if p.Websites == nil {
		p.Websites = []model.Website{}
	}
	return nil
}
