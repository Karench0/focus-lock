//go:build windows

package desktop

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const KotlinDocsURL = "https://kotlinlang.org/docs/home.html"

// OpenKotlinDocs opens only the official documentation through the user's
// current default HTTPS handler. No browser settings or associations change.
func OpenKotlinDocs() error {
	verb, _ := windows.UTF16PtrFromString("open")
	target, _ := windows.UTF16PtrFromString(KotlinDocsURL)
	if err := windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("Не удалось открыть документацию Kotlin / Could not open Kotlin documentation: %w", err)
	}
	return nil
}
