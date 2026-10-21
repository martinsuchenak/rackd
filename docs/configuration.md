# Configuration

Rackd's server is configured with environment variables (optionally from a `.env` file). Every setting, its default and how values are loaded and checked are in the [Configuration Reference](configuration-reference.md).

- Authentication needs no settings: people sign in with user accounts and scripts use API keys. See [Authentication](authentication.md).
- The first admin account comes from `INITIAL_ADMIN_USERNAME` and `INITIAL_ADMIN_PASSWORD` on first start.
- Set `ENCRYPTION_KEY` (64 hex characters, `openssl rand -hex 32`) to enable credentials, scan profiles, scheduled scans and DNS.
- CLI settings (`RACKD_SERVER_URL`, `RACKD_TOKEN`) are covered in [CLI](cli.md).
- Systemd, Docker and Nomad examples are in [Installation](installation.md) and [Deployment](deployment.md).
