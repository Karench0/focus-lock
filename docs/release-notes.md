# Focus Lock 0.1.0 — Windows preview

The first public preview includes the Go/Wails Windows app, Russian/English graphite UI, application and website profiles, backend session/exit timers, SQLite statistics, tray lifecycle, and a Chrome/Edge Manifest V3 companion extension.

Download the **Windows x64 ZIP** and extract everything together, or use the per-user **setup executable**. The GUI requires WebView2. For website filtering, connect the local bridge in Settings and load the bundled `browser-extension` folder in your browser's Developer mode. The extension is not in a browser store yet.

**This is a prerelease, not a certified lockdown tool.** Process monitoring is reactive, excludes protected/elevated/system processes and existing background programs, and can be bypassed by the computer's owner. New disallowed processes may lose unsaved work when terminated. Save work before use. Browser-profile/incognito coverage, hardware suspend/reboot and a broad Windows app compatibility matrix remain stable-release gates. See `docs/testing.md` and the README.

All restrictions are intended to end with the GUI process. Task Manager stays available. The extension uses session-only rules, a native disconnect callback and a short lease; browser scheduling can delay cleanup. Disable the extension or restart the browser if needed. The app does not change hosts, firewall or proxy rules.

Executables are **unsigned**. SHA-256 hashes are included for downloaded-file integrity. Profiles and statistics stay local; visited URLs and page contents are not stored.
