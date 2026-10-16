//go:build windows

package applications

import (
	"os"
	"testing"
)

// This opt-in check only reads associations and the browser image. It neither
// launches the browser nor modifies the user's Windows settings.
func TestDefaultBrowserSmoke(t *testing.T) {
	if os.Getenv("FOCUS_LOCK_DEFAULT_BROWSER_SMOKE") != "1" {
		t.Skip("set FOCUS_LOCK_DEFAULT_BROWSER_SMOKE=1 on a Chrome/Edge desktop")
	}
	app, err := DefaultBrowser()
	if err != nil {
		t.Fatal(err)
	}
	if app.Name == "" || app.Path == "" || (Browser(app.Path) != "chrome" && Browser(app.Path) != "edge") {
		t.Fatalf("invalid supported default browser: %+v", app)
	}
	t.Logf("Verified default browser: %s", app.Name)
}
