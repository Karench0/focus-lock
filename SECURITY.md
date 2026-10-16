# Security policy

Focus Lock is a voluntary productivity tool, not a security boundary. Deliberate user bypass is expected; safety, reliable release of restrictions, and privacy are the priorities.

Report a vulnerability using the repository's **Security → Report a vulnerability** feature when available. If private reporting is unavailable, open an issue with a brief description and request a private reporting channel before sharing sensitive details. Do not include tokens, private paths, page contents, or personal browsing history in a public report.

Include the app version, Windows build, browser version, whether the extension was installed, steps to reproduce, and whether terminating the GUI or restarting the browser restored access. Treat unexpected system-process termination, a channel-authentication bypass, stored personal browsing data, and restrictions surviving app/browser exit as security-relevant bugs.

Only the current prerelease branch is supported. There is no guaranteed response SLA or digitally signed distribution yet. Dependencies are pinned and checked by CI. No password, API token or remote control endpoint is required for app operation.

Emergency recovery: end `FocusLock.exe` in Task Manager. If browser scheduling delays extension cleanup, disable the Focus Lock extension or restart that browser. No `hosts`, firewall or proxy restoration should be necessary. Keep a reproducer before reporting the issue, without exposing private data.
