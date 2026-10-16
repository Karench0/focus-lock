# Focus Lock 0.3.0 — Windows preview

This version adds the explicitly confirmed personal Storozhenko Kotlin exercise. Settings has a timed triple-activation circle. The exercise temporarily replaces website/app preferences, opens official Kotlin docs in the default supported browser, and ends after one correct eight-option Kotlin question. It preserves saved preferences and never auto-starts after restarting the app. Update/reload the bundled extension before use.

A compact timer sits above other windows during a session. Click **Mini timer**, move it by its grip, or expand back to the original window. Stop uses confirmation and the existing delay in ordinary sessions; in the untimed personal exercise it opens the Kotlin question. Resize takes 240 ms and respects reduced motion. The timer can stay compact when a session finishes.

The preceding update fixes missing website styles/scripts in strict mode and the extension-folder button. In Websites, expand Resource domains and explicitly approve the required CDN domains. GitHub has a suggested list. Resource permissions apply only to requests originating from the selected site; direct navigation and embedded frames remain blocked. Update the desktop app and reload the unpacked extension, approve resources before starting a new session, then reload the website. Existing profiles are preserved.

The public preview includes the Go/Wails Windows app, Russian/English graphite UI, application and website profiles, backend session/exit timers, SQLite statistics, tray lifecycle, and a Chrome/Edge Manifest V3 companion extension.

Download the **Windows x64 ZIP** and extract everything together, or use the per-user **setup executable**. The GUI requires WebView2. For website filtering, connect the local bridge in Settings and load the bundled `browser-extension` folder in your browser's Developer mode. The extension is not in a browser store yet.

**This is a prerelease, not a certified lockdown tool.** Process monitoring is reactive, excludes protected/elevated/system processes, and can be bypassed by the computer's owner. Ordinary sessions exempt existing background programs. The personal exercise also closes unrelated existing programs after explicit review and consent, and replaces unrelated open HTTP(S) tabs. Terminated programs may lose unsaved work. Save work before use. Browser-profile/incognito coverage, hardware suspend/reboot and a broad Windows app compatibility matrix remain stable-release gates. See `docs/testing.md` and the README.

All restrictions are intended to end with the GUI process. Task Manager stays available. The extension uses session-only rules, a native disconnect callback and a short lease; browser scheduling can delay cleanup. Disable the extension or restart the browser if needed. The app does not change hosts, firewall or proxy rules.

Executables are **unsigned**. SHA-256 hashes are included for downloaded-file integrity. Profiles and statistics stay local; visited URLs and page contents are not stored.
