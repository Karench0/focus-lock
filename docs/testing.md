# Verification and remaining release gates

Recorded on 2026-10-16 on Windows 11 Pro, build 10.0.26300. This report distinguishes component tests from manual Windows acceptance. Version 0.1.0 is a prerelease because the remaining hardware/profile compatibility gates have not all been exercised.

## Automated checks completed locally

- TypeScript strict check and production Vite build.
- ESLint with zero warnings.
- Seven Node rule/lease/framing-policy tests, including exact domain suffix/credential tricks, explicit subdomains/localhost, strict main-frame allow rules, blocklist behavior and rule removal.
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
