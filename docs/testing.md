# Verification and remaining release gates

Recorded on 2026-10-16 on Windows 11 Pro, build 10.0.26300. This report distinguishes component tests from manual Windows acceptance. Versions through 0.3.0 remain previews because the remaining hardware/profile compatibility gates have not all been exercised. Update-specific evidence appears below.

## Automated checks completed locally

- TypeScript strict check and production Vite build.
- ESLint with zero warnings.
- Seventeen Node rule/lease/framing-policy and exercise-tab tests, including exact domain suffix/credential tricks, explicit subdomains/localhost, strict main-frame allow rules, blocklist behavior and rule removal.
- Go unit/integration tests with the race detector: normal completion, early-exit delay, zero delay, cancellation, repeated exit request, late scheduler/earliest deadline, concurrent finish idempotency, invalid durations, profile/domain validation, storage idempotency, crash-record recovery, statistics and current-user process identity checks.
- A real, disposable child executable was inspected and terminated; a stale creation timestamp was refused. No unrelated desktop process was used as a blocking test fixture.
- Backend service integration rejects profile/settings/deletion mutations while active.
- Go vet and a compiled Windows x64 GUI plus native host.
- A real Wails desktop window was launched and observed with its Russian onboarding and application selection.
- Playwright ran the compiled GUI in explicit diagnostic-preview mode against real SQLite and Go timers: onboarding, profile edits, website edits, active session, early exit, cancellation, a real 15-second delay, and a real 60-second natural completion. It checked statistics and recorded six UI screenshots with no page errors. **Preview mode intentionally disables process enforcement.**
- An isolated Chrome for Testing 151.0.7922.34 profile loaded the actual unpacked extension. A test-only Go bridge fixture used the production named-pipe/authentication implementation and the compiled native host. The browser acknowledged rules, allowed a listed localhost page, redirected an unlisted loopback page, blocked an unlisted network subresource, and passed blocklist checks. Killing the bridge fixture removed browser rules and restored access in **59 ms in the recorded run**. This is an observation, not a timing guarantee. Restarting the browser without the app left no session rules.
- The extension test temporarily registered its native host under HKCU and restored the previous entries in `finally`. It used a unique pipe suffix and never controlled desktop applications.
- The same actual navigation, blocklist, native-channel, crash cleanup and browser-restart checks also passed in an isolated Microsoft Edge **155.0.4283.45** profile. The recorded restoration after killing the bridge fixture was 177 ms in that Edge run.
- `npm audit` found zero known vulnerabilities after the dependency updates used for the recorded build.

## Required scenario matrix

| Scenario | Evidence / status |
| --- | --- |
| Disallowed application | Policy tests plus real disposable-process termination test; full mixed desktop compatibility remains manual |
| Allowed application | Full-path and child-policy unit tests; real app compatibility remains manual |
| Disallowed website | Actual Chromium DNR main-frame redirect and subresource block passed |
| Allowed website | Actual Chromium allowlisted localhost navigation passed |
| Profile changes | SQLite, backend and UI end-to-end checks passed |
| Main session ends | Injected-clock tests and real 60-second Go session passed |
| Request early exit | Backend and UI checks passed |
| Cancel early exit | Backend and UI checks passed |
| Exit delay expires | Injected-clock and real 15-second UI checks passed |
| Close window / tray | Native app launch observed; active-window close, tray restore and OS notification need manual acceptance |
| Task Manager termination | Production bridge implementation tested by forcibly killing its isolated fixture; active full GUI/desktop session remains manual |
| Crash | Browser fail-open fixture and SQLite unfinished-row recovery passed |
| Reboot | No durable enforcement exists and storage restarts idle; physical Windows reboot remains untested |
| Missing extension | Policy rejects unverified known browsers; disconnected state shown in UI; browser restart without app passed |

## Reproduce browser integration

Build the app as described in the README, then:

```powershell
cd tests
npm ci
npx playwright install chromium
npm run ui
cd ..
go build -o build/bin/bridge-fixture.exe ./internal/bridge/testdata/server
cd tests
npm run extension
# Also test the installed Microsoft Edge in an isolated profile:
$env:FOCUS_LOCK_TEST_BROWSER='msedge'
npm run extension
```

The extension test needs permission to temporarily write current-user Native Messaging registrations. It backs up existing values and restores them. The GUI does not need to be closed because the fixture uses an isolated named pipe. Only test-created browser profiles and child processes are used. Set `FOCUS_LOCK_SCREENSHOTS=1` to capture UI screens; screenshot history reflects actual test sessions, not seeded focus statistics.

## Stable-release gates

Do not tag 1.0.0 until these checks are recorded on Windows 11 with current Chrome and Edge:

1. Install/uninstall and launch the packaged build on a clean, non-development Windows account, including missing-WebView2 behavior.
2. Save/close preflight behavior and repeated enforcement with editors, Office, PDF readers, terminals, games and their legitimate helpers.
3. Confirm that Windows security, accessibility software, screen readers, drivers and services remain usable.
4. Active-window X, tray hide/restore, completion notifications, delayed quit, Task Manager termination and forced GUI crash.
5. Suspend/resume, clock adjustment, physical reboot and session recovery.
6. Chrome and Edge, multiple browser profiles/roots, extension disable/update, guest/incognito restrictions and service-worker restarts.
7. Verify local files, browser caches/service-worker responses, existing tabs and authentication/CDN dependencies are described accurately.

CI reruns automated checks on a Windows runner. A green CI result alone does not satisfy the manual/hardware gates above.

## Compact window verification — 2026-10-16

UI E2E covers entering/expanding the mini timer, confirmation without stopping, exit countdown/cancel without changing session ID, natural completion inside the mini timer, and viewport overflow at 420 × 180. Geometry tests cover restoring bounds after dragging, duplicate requests, maximised state, animation monotonicity and reduced motion.

For manual native checks, use a separate `FOCUS_LOCK_DATA_DIR` and launch `FocusLock.exe --window-preview`. This diagnostic mode disables process/website enforcement and bypasses the production instance lock; it is for UI checks only. Check native dragging, maximise → compact → restore, tray hide/Open, Alt+F4 while compact, and 100/125/150% DPI. Automated tests do not replace the remaining hardware/multi-monitor gates above.

Native Windows preview results: compact resizing reached 420 × 180; pointer dragging moved the window from (958, 189) to (1146, 288) at the current 125% display scale; expanding restored (958, 189) and the original size. Maximise/compact/expand was also checked. The session continued throughout. Mixed-DPI physical monitor changes remain a manual compatibility check.

Alt+F4 while compact was verified in the native preview: the window expands and shows the existing active-session close dialog. Window API calls are dispatched after the close callback returns to avoid blocking the Windows UI thread.

## Personal Kotlin exercise verification — 2026-10-16

Go race tests cover explicit consent, expiring preparation tokens, temporary profiles, browser-root-specific extension capability, start cancellation, untimed exercise clocks, and single-use quiz answers. The question pool contains 20 distinct questions with eight options each; answer keys stay in the Go backend. Policy tests verify that only the selected IDE's bundled Java helpers receive additional permission. Go vet, frontend type checking, lint and production builds passed.

The compiled GUI's isolated diagnostic preview passed the ordinary session regression suite and the exercise end-to-end suite: slow clicks do not reveal the panel, a timed triple activation does, consent can be cancelled, compact Stop opens the quiz, wrong answers keep the same session running, replayed answers fail, and a correct answer ends the exercise. Saved preferences and subsequent normal sessions remain intact. At 1000 px and 730 px window heights the eight answers scroll while the action buttons remain visible; keyboard selection and submission also passed.

All 17 extension unit tests passed. Actual Native Messaging and DNR integration passed in isolated Chrome for Testing and Microsoft Edge profiles. Exercise start replaced already-open unrelated HTTP(S) tabs while retaining allowed tabs; normal sessions left existing tabs alone. A live official Kotlin documentation page loaded its styles successfully in Chrome. Killing the isolated bridge fixture restored access in 134 ms (Chrome) and 146 ms (Edge) in the recorded runs; these are observations, not guarantees. Browser restart also cleared session rules. Temporary current-user host registrations were restored after testing.

Default-browser detection was checked read-only. The real all-application exercise was not activated on the user's desktop: GUI tests used preview mode, browser tests used disposable profiles, and process-control checks used policy tests and the existing disposable-child termination test. Broad desktop compatibility and the hardware gates above remain manual checks.

## Protected-process regression — 2026-10-17

The full Go race suite, Go vet, frontend type checking and production builds passed after the 0.3.1 fix. A synthetic closure sequence makes DefenderSessionHelper return wrapped access-denied, missing-privilege, or protected-process errors before Steam and Telegram; each case keeps the exercise active, closes both remaining synthetic apps, remembers the skipped instance, and counts only successful closures. The same error classification is used by active monitoring. Unexpected errors remain reportable. Security-name checks exclude DefenderSessionHelper while retaining ordinary Steam/Telegram/Dota restrictions. The real owned disposable-child test passed, rejecting protected and stale identities and terminating only its own fixture. No actual security or user applications were closed to test this fix.

## Startup, page dependencies and code formatting — 2026-10-17

The 0.3.2 extension has 23 passing unit tests, including native heartbeats during deferred DNR installation and tab APIs, prompt reconnection, cancellation before acknowledgement, and fail-open cleanup while a tab API remains pending. Go regressions verify that heartbeats and block reporting do not wait for the service mutex, restarted instances of reviewed executable paths are accepted while different paths are refused, and old extensions cannot report readiness. Protected and already-exited process instances do not interrupt the remaining closure sequence. Actual full-desktop enforcement was not activated on the user's applications.

Isolated Chrome and Edge integration passed five complete start/stop cycles with cache disabled. A synthetic page loaded a cross-domain stylesheet, imported stylesheet, script, image and JSON API with no resource-domain entries. Unrelated initiators, separate navigation and unlisted frames remained blocked. Both browsers completed a deliberately delayed 8.5-second tab replacement on the same native connection, acknowledged only after completion, and cleared restrictions after killing the bridge (140 ms in Chrome and 156 ms in Edge in these runs). Temporary HKCU registrations were restored.

Live public GitHub, Kotlin documentation and Yandex Music pages each passed two cold loads in isolated Chrome: stylesheets loaded and no page resource request was blocked by the extension. Public-page network or third-party reporting errors are not evidence of account-specific playback or sign-in compatibility; no personal browser profile or account was used.

The compiled GUI preview passed failed-start recovery with a fresh token and renewed consent, compact Stop, wrong/correct answers, and formatted Kotlin examples. Code preserves whitespace and four-space indentation with syntax highlighting. Keyboard navigation and action-button visibility passed at 1000 px and 730 px window heights. Examples and the source-only integration oracle were updated together; correct answers stay in the Go backend.
