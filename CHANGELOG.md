# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Changes before v0.0.20 are recorded in the [git history](https://github.com/martinsuchenak/rackd/commits/main)
and [GitHub releases](https://github.com/martinsuchenak/rackd/releases).

## [Unreleased]

## [0.0.21] - 2026-10-13

### Added

- This changelog. Each release's section is published as its GitHub release
  notes and on the [documentation site](https://getrackd.dev/changelog/).

### Changed

- RBAC permission checks now use the [go-rbac](https://github.com/martinsuchenak/go-rbac)
  engine (v0.1.0). Role and permission definitions are kept in memory and reloaded
  whenever a role or permission changes; permissions still match exactly
  (wildcard grants are disabled).
- The permissions reported for a user (UI configuration, OAuth scope checks) are now
  derived from the same source as enforcement, so they always agree.
- DNS provider (Technitium) calls and webhook test pings now time out after 20s
  instead of 30s, so failures are reported before the request timeout.
- The network topology graph uses Cytoscape's default scroll-zoom sensitivity
  (removes the "custom wheel sensitivity" console warning).
- Updated Go dependencies (including `paularlott/mcp` v0.32.4, `paularlott/cli`
  v0.10.1, `modernc.org/sqlite` v1.60.1, `golang.org/x/crypto` v0.58.0) and web UI
  dependencies (Alpine.js 3.17.4, Cytoscape 3.34.3, Playwright 1.64.0). Run
  `bun run test:e2e:install` to fetch the matching Playwright browser.

### Fixed

- Requests running longer than 15s received an empty reply because the HTTP server
  write timeout was shorter than `REQUEST_TIMEOUT`. The write timeout now outlasts
  the request timeout (`REQUEST_TIMEOUT` + 5s).
- DNS provider "Test Connection" failures against unreachable servers are shown
  again (502 `PROVIDER_UNREACHABLE` with the cause) instead of hanging.
- OAuth client registration without a `scope` returns 400
  `invalid_client_metadata` instead of a 500 server error.
- Approving an OAuth authorization whose requested scopes are not permitted for the
  user returns `invalid_scope` instead of `server_error`.
- Deleting a permission now revokes it immediately rather than at the next role
  change or restart.

### Removed

- Internal `HasPermission`, `HasAnyPermission` and `HasAllPermissions` helpers,
  replaced by go-rbac.

## [0.0.20] - 2026-09-19

### Changed

- Build tooling moved from a Makefile to a Taskfile.
- Go bumped to 1.27.1 and dependencies updated.
- Tests can lower the bcrypt cost via `RACKD_BCRYPT_COST` for faster runs.

### Fixed

- Resolved all golangci-lint findings.

[Unreleased]: https://github.com/martinsuchenak/rackd/compare/v0.0.21...HEAD
[0.0.21]: https://github.com/martinsuchenak/rackd/compare/v0.0.20...v0.0.21
[0.0.20]: https://github.com/martinsuchenak/rackd/compare/v0.0.19...v0.0.20
