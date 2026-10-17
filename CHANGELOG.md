# Changelog

## 0.3.2 — 2026-10-17 (prerelease)

- Allow page-initiated CSS, scripts, fonts, images, API calls and media automatically while keeping other website destinations and unlisted frames restricted.
- Keep native heartbeats independent of long process/discovery work and asynchronous browser rule/tab operations; reconnect promptly after transient disconnects and cancel pending tab sweeps on stop/disconnect.
- Permit restarted instances of reviewed executable paths, skip already-exited processes, refresh failed start tokens, wait up to 15 seconds for exercise acknowledgement, and indicate outdated extensions.
- Format question examples as indented Kotlin code blocks with syntax highlighting and keyboard-accessible scrolling.

## 0.3.1 — 2026-10-17 (prerelease)

- Exempt DefenderSessionHelper and Defender platform directories, elevated processes, and processes without termination permission from the closure review.
- An access-denied or protected-process result during exercise startup or monitoring now skips that process and continues closing ordinary disallowed applications. The skipped instance is remembered without counting it as closed. Unexpected failures still report an error.
- Regression coverage for protection changes after consent, continued Steam/Telegram closure, and owned disposable-process safety.

## 0.3.0 — 2026-10-16 (prerelease)

- Hidden personal Storozhenko mode behind a timed triple activation in Settings, with explicit identity/consent confirmation.
- An untimed Kotlin-only exercise uses a temporary policy and preserves normal profiles/settings. Existing unrelated apps and web tabs are reviewed before enabling; default Chrome/Edge must have an updated, connected extension.
- Opens official Kotlin documentation in the default browser; permits its scoped font resources and installed IntelliJ IDEA/Android Studio helpers.
- Twenty distinct middle-level Kotlin questions with eight shuffled options, one-use challenges and backend answer validation. Wrong answers keep the exercise active; a correct answer finishes it.
- Normal early exit cannot bypass the exercise. Closing or crashing the app still releases restrictions; no exercise automatically restarts.

## 0.2.0 — 2026-10-16 (prerelease)

- Compact, always-on-top 420 × 180 timer with native dragging, progress, stop confirmation and an inline early-exit countdown.
- A 240 ms eased window transition restores the original position, size and maximised state; reduced motion skips animation.
- Custom window controls, tray restore and a quiet compact completion state. Session timing and restrictions continue unchanged.
- Compact-flow and geometry regression checks, plus native Windows window-preview mode for safe manual checks.

## 0.1.1 — 2026-10-16 (prerelease)

- Explicit per-site resource domains restore CDN styles, scripts and images in strict mode without permitting navigation to those domains.
- GitHub resource suggestions require approval; existing profiles retain their permissions.
- Open extension folder uses Windows Explorer and reports missing files.
- Regression coverage for resource isolation, profile persistence and folder resolution.

## 0.1.0 вЂ” 2026-10-16 (prerelease)

- Initial Windows/Wails application with a graphite UI, six screens, Russian/English text and onboarding.
- SQLite profiles, settings and recoverable local session history.
- Backend-owned session and early-exit deadlines with race/idempotency tests.
- Same-user Windows process discovery, safety exemptions, full-path allowlists and opt-in children.
- Chromium MV3 website rules, exact/subdomain matching and Native Messaging fail-open leases.
- Tray lifecycle, preflight warnings, diagnostic preview, Windows build and release automation.
- Portable packaging, per-user Inno Setup configuration, original logo and real UI captures.

This is a preview release. Hardware reboot/suspend, broad app compatibility and all browser-profile combinations remain stable-release gates; see `docs/testing.md`.
