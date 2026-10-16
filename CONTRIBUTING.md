# Contributing

Use Windows 11 x64, Go 1.25.6+ and Node 24. Follow the README development instructions. Keep changes small and describe the observable behavior and the checks you ran. Run TypeScript, ESLint, extension tests, Go tests with the race detector, `go vet`, and a full Windows build before requesting review.

Go files must be formatted with `gofmt`. Keep domain validation, timer state and safety decisions in Go; the UI is not trusted to enforce them. Test deadlines with an injected clock rather than real delays. Process tests must launch and terminate only their own disposable fixtures. Browser integration tests must use an isolated temporary profile. Never run a real strict session on another person's desktop as an automated test.

Do not add telemetry, cloud dependencies for local operation, persistent OS restriction rules, kernel drivers, process injection, or Task Manager interception. New exemptions should explain the Windows component they protect. Add explicit documentation when enforcement cannot reliably cover a case.

Use imperative commit subjects. PRs should describe the problem, resulting behavior, relevant limitations and validation. Include real screenshots for visual changes, with personal file paths and content removed. Do not commit local data, generated executables, tokens, dependencies or environment files.

Releases are created by a maintainer pushing a version tag after CI and the release gates in `docs/testing.md` are satisfied. Pre-1.0 tags are prereleases. A stable tag must not be used merely to indicate that compilation succeeded.
