# Architecture

## Boundaries

`main.App.Call` is the Wails command boundary. It decodes JSON into explicit Go types and delegates validation to `internal/api.Service`. Unknown commands fail. Profile and settings mutations are serialized with session start under the same mutex; direct calls cannot bypass the UI lock. The React UI polls authoritative state once a second. The diagnostic preview is a separate, explicit mode: an authenticated random-port loopback API with process enforcement disabled.

The service owns a session engine, SQLite store, a 700 ms enforcement loop and a named-pipe browser bridge. The engine has its own mutex and injectable clock. Deadlines use Go `time.Time` monotonic data in memory. Serialized timestamps are informational and for recovery; they are never used to resume restrictions. Tied natural and early-exit deadlines prefer natural completion. Finish events are emitted once; database updates additionally require an active record.

## Windows process policy

The toolhelp process snapshot is enriched using limited-information handles, full executable paths, creation timestamps, tokens and session IDs. Inaccessible, elevated, critical, system-path, foreign-user, foreign-session, UIAccess accessibility processes and known security/accessibility tools are protected. Registered antivirus installation directories are read from Windows Security Center, with conservative vendor-directory fallbacks if WMI is unavailable. The GUI, native host and WebView2 are exempt. User programs are allowlisted by canonical full path; no wildcard application rule is supported. A single reusable EnumWindows callback avoids accumulating non-freeable Windows callbacks over long sessions.

The preflight reports disallowed visible programs and unverified known browser processes. It refuses start until those are closed or permitted. Existing non-browser background processes form a baseline and are not terminated. New non-exempt processes are evaluated against the immutable session profile. A failed termination is reported in the UI rather than retried in a tight loop.

Immediately before termination the app checks process creation time on the handle that will be used for termination, then rechecks safety/path. This avoids authorizing an unrelated process after PID reuse. There is no code injection, kernel interception, privileged service, global keyboard hook, hosts edit or durable OS restriction policy.

Child permission is explicit and requires an ancestor whose path is allowlisted plus a child path within its installation directory. Browsers never confer permission to arbitrary children. This is compatibility support, not a secure sandbox; applications capable of running arbitrary code inherently permit bypass.

## Browser protocol

```text
MV3 worker -- length-prefixed JSON stdio --> FocusLock-host.exe
            -- current-user ACL named pipe --> GUI bridge
```

The unpacked extension's public key fixes its ID. Chrome/Edge restrict Native Messaging launch to that allowed origin. The host independently verifies the origin argument. The pipe server verifies the actual client process image against the companion host path and identifies its browser ancestor from a Windows process snapshot. A different-user process cannot open the pipe. This does not defend against an administrator or a hostile program that can replace files as the same user.

The extension requests rules and sends `{ok, applied, blocks, studyCapable, resourceCapable}` every two seconds, independently of asynchronous DNR installation and tab replacement. Only an actual app response renews its six-second lease. The server reads engine state and cached language without the service mutex or SQLite, so discovery and process control cannot stall heartbeats. A successful DNR update and completed exercise tab sweep set the applied session ID; obsolete/cancelled sweeps cannot acknowledge a session. A normal strict session waits up to five seconds for browser acknowledgement; an exercise allows 15 seconds. Native parent PID and creation time identify the verified browser root. Browser profiles sharing one root cannot be fully distinguished: this is a documented preview limitation.

The rules compiler emits a low-priority main-frame redirect and a network-subresource block for strict mode, with higher-priority exact-host allows. Resource requests initiated by an allowed domain also receive an allow rule excluding main_frame and sub_frame, so page assets/APIs/media do not need manually enumerated CDN hosts. Legacy explicit resource entries remain valid. This trusts an allowed site's resource requests, not arbitrary navigation or third-party embedded documents. Blocklist mode emits rules only for chosen domains. Domain input is normalized with IDNA in Go; the worker validates its own input again. Permissions cover all hosts so it can filter either mode. It does not send full URLs to Go. The blocked page reports a count only.

## Exit and failure cases

| Event | Result |
| --- | --- |
| Natural timer expiry | Engine finishes; monitor stops; next bridge update clears DNR |
| Early exit requested | Immutable exit deadline; restrictions and main timer continue |
| Early exit cancelled | Exit timestamp cleared; original session end retained |
| Window X while active | UI chooses tray or early-exit procedure |
| Tray chosen | Only window hidden; Go and native bridge remain alive |
| Normal inactive quit | Store closed; pipe disconnected; tray removed |
| Task Manager / crash | Monitor dies with GUI; OS closes pipe; native host exits; worker clears rules |
| Lost native connection | Worker clears rules; main app stops treating browser as verified |
| Worker restart | Clears old session rules before reconnecting |
| Browser restart | DNR session rules disappear; worker starts fail-open |
| Reboot / app restart | SQLite marks unfinished rows recovered; engine starts idle |

Native connection lifetime keeps the MV3 worker alive during normal operation. A short timer expires its lease; a 30-second alarm is a fallback when a worker wakes later. Browser/OS suspension can delay cleanup, so fail-open timing is best effort rather than a real-time promise. Disabling the extension or restarting the browser is always an emergency release.

## Data

SQLite contains profile JSON, settings JSON and session aggregates. WAL and busy timeout reduce contention. Active records are checkpointed every three seconds; recovery counts only persisted time. There is no URL history, page content, text capture, account, telemetry or cloud sync. The SQL source is embedded from `internal/storage/schema.sql`.

## Distribution

`scripts/build.ps1` builds the GUI, separate console native host and companion extension. `scripts/package.ps1` creates a complete portable ZIP and optional non-admin Inno Setup installer with SHA-256 hashes. Tag pushes run Windows tests/build before publishing. Pre-1.0 tags are prereleases. No code-signing claim is made.
