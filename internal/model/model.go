package model

type Application struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Icon          string `json:"icon,omitempty"`
	AllowChildren bool   `json:"allowChildren"`
}

type Website struct {
	Domain     string   `json:"domain"`
	Subdomains bool     `json:"subdomains"`
	Resources  []string `json:"resources,omitempty"`
}

type Profile struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Applications []Application `json:"applications"`
	Websites     []Website     `json:"websites"`
	Mode         string        `json:"mode"`
}

type Settings struct {
	Language         string `json:"language"`
	ExitDelaySeconds int    `json:"exitDelaySeconds"`
	Onboarded        bool   `json:"onboarded"`
}

type Record struct {
	ID         string `json:"id"`
	Profile    string `json:"profile"`
	StartedAt  string `json:"startedAt"`
	EndedAt    string `json:"endedAt"`
	Seconds    int    `json:"seconds"`
	Outcome    string `json:"outcome"`
	AppBlocks  int    `json:"appBlocks"`
	SiteBlocks int    `json:"siteBlocks"`
}

type Day struct {
	Date    string `json:"date"`
	Seconds int    `json:"seconds"`
}

type Statistics struct {
	Seconds     int      `json:"seconds"`
	Completed   int      `json:"completed"`
	Interrupted int      `json:"interrupted"`
	AppBlocks   int      `json:"appBlocks"`
	SiteBlocks  int      `json:"siteBlocks"`
	Days        []Day    `json:"days"`
	History     []Record `json:"history"`
}
