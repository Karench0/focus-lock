package applications

import (
	"github.com/Karench0/focus-lock/internal/model"
	"testing"
)

func TestAllowAndBlockByCanonicalPath(t *testing.T) {
	rules := []model.Application{{Path: `C:\Tools\Code\Code.exe`}}
	for _, tt := range []struct {
		p    Process
		want bool
	}{{Process{Path: `c:\tools\code\code.exe`}, true}, {Process{Path: `C:\Games\Code.exe`}, false}, {Process{Path: `C:\Windows\system.exe`, Safe: true}, true}} {
		if got := Allowed(tt.p, nil, rules, true, nil); got != tt.want {
			t.Fatal(tt, got)
		}
	}
}
func TestChildrenRequireOptInAndInstallationBoundary(t *testing.T) {
	root := Process{PID: 10, Path: `C:\Tools\Editor\editor.exe`}
	child := Process{PID: 11, Parent: 10, Path: `C:\Tools\Editor\bin\helper.exe`}
	all := []Process{root, child}
	rules := []model.Application{{Path: root.Path, AllowChildren: true}}
	if !Allowed(child, all, rules, false, nil) {
		t.Fatal("required child blocked")
	}
	child.Path = `C:\Other\helper.exe`
	if Allowed(child, all, rules, false, nil) {
		t.Fatal("outside installation inherited")
	}
	rules[0].AllowChildren = false
	child.Path = `C:\Tools\Editor\helper.exe`
	if Allowed(child, all, rules, false, nil) {
		t.Fatal("child requires opt in")
	}
}
func TestStrictBrowserRequiresHealthEvenWhenWhitelisted(t *testing.T) {
	p := Process{Path: `C:\Chrome\chrome.exe`}
	rules := []model.Application{{Path: p.Path}}
	if Allowed(p, nil, rules, true, nil) {
		t.Fatal("unverified browser allowed")
	}
	if !Allowed(p, nil, rules, true, func(Process, []Process) bool { return true }) {
		t.Fatal("verified browser blocked")
	}
	p.Path = `C:\Firefox\firefox.exe`
	rules[0].Path = p.Path
	if Allowed(p, nil, rules, true, nil) {
		t.Fatal("unsupported browser allowed")
	}
}
func TestBrowserCannotAuthorizeDownloadedChildren(t *testing.T) {
	root := Process{PID: 10, Path: `C:\Chrome\chrome.exe`}
	p := Process{PID: 11, Parent: 10, Path: `C:\Chrome\game.exe`}
	if Allowed(p, []Process{root, p}, []model.Application{{Path: root.Path, AllowChildren: true}}, true, nil) {
		t.Fatal("browser child inherited permission")
	}
}
