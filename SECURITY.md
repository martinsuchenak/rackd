# Security Policy

## Supported versions

Rackd is pre-1.0. Security fixes are made on `main` and shipped in the next release; only the latest release is supported.

## Reporting a vulnerability

Please do not open a public issue for a vulnerability.

Report it privately through [GitHub security advisories](https://github.com/martinsuchenak/rackd/security/advisories/new). If that page is not available, open an issue that asks for a private contact, without any details of the vulnerability.

Include the affected version, how to reproduce the issue, and its impact as you understand it.

## Disclosure

You are credited in the advisory and release notes unless you ask not to be. Please give us a chance to release a fix before disclosing the issue publicly.

## Verifying releases

Release checksums and SBOMs are signed with [cosign](https://github.com/sigstore/cosign) using GitHub's OIDC identity, and each release carries a build provenance attestation. See [Verifying a release](docs/installation.md#verifying-a-release).
