# Contributing to Rackd

Thanks for helping improve Rackd.

## Before you start

- For anything larger than a small fix, open an issue first so the approach can be agreed.
- Rackd is licensed under the [Apache License 2.0](LICENSE). Your first pull request asks you to sign the [Contributor License Agreement](CLA.md) by posting a comment. One signature covers all future contributions.

## Development

See the [Development Guide](docs/development.md) for building and running Rackd. Before opening a pull request:

```bash
task vet
task lint
task test
cd webui && bun run typecheck && bun run validate:types && bun test
```

CI runs the same checks, plus the tests with the race detector, on every pull request.

## Pull requests

- Keep each pull request to one change, with tests.
- Add an entry under `## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md) for anything users will notice.
- Schema changes go in a new migration in `internal/storage/migrations.go` with a working Down step. The reversibility test checks that every migration undoes cleanly.

## Reporting security issues

Do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md) for how to report one privately.
