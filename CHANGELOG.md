# Changelog

## 0.1.1 — 2026-10-16 (prerelease)

- Explicit per-site resource domains restore CDN styles, scripts and images in strict mode without permitting navigation to those domains.
- GitHub resource suggestions require approval; existing profiles retain their permissions.
- Open extension folder uses Windows Explorer and reports missing files.
- Regression coverage for resource isolation, profile persistence and folder resolution.

## 0.1.0 â€” 2026-10-16 (prerelease)

- Initial Windows/Wails application with a graphite UI, six screens, Russian/English text and onboarding.
- SQLite profiles, settings and recoverable local session history.
- Backend-owned session and early-exit deadlines with race/idempotency tests.
- Same-user Windows process discovery, safety exemptions, full-path allowlists and opt-in children.
- Chromium MV3 website rules, exact/subdomain matching and Native Messaging fail-open leases.
- Tray lifecycle, preflight warnings, diagnostic preview, Windows build and release automation.
- Portable packaging, per-user Inno Setup configuration, original logo and real UI captures.

This is a preview release. Hardware reboot/suspend, broad app compatibility and all browser-profile combinations remain stable-release gates; see `docs/testing.md`.
