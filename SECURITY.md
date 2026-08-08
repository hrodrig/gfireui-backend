# Security Policy

## Scope

This policy covers the **gfireui-backend** Go BFF binary, its configuration
handling, Postgres-backed stores, the HTTP API, and published container images
(`ghcr.io/hrodrig/gfireui-backend`).

The browser SPA lives in **[gfireui](https://github.com/hrodrig/gfireui)** —
report UI vulnerabilities there. Engine issues belong in
**[gfire](https://github.com/hrodrig/gfire)**. Deployment packaging belongs in
**[gfire-selfhosted](https://github.com/hrodrig/gfire-selfhosted)**.

## Supported Versions

We support the **latest release** and the active development branch with
security updates. We use [semantic versioning](https://semver.org/)
(MAJOR.MINOR.PATCH).

| Version | Supported |
| ------- | --------- |
| Latest release | :white_check_mark: |
| Older releases | :x: |

When a vulnerability is fixed, we release a new patch version. Please upgrade
to the latest release to receive security fixes.

## Reporting a Vulnerability

**Do not open a public issue** for security vulnerabilities.

- **Preferred:** Use [GitHub Security Advisories](https://github.com/hrodrig/gfireui-backend/security/advisories/new) to report privately.
- **Alternative:** Contact the maintainer via [github.com/hrodrig](https://github.com/hrodrig) with:
  - clear description
  - impact
  - steps to reproduce
  - affected versions / image tags (if known)

## What to expect

- We acknowledge your report as soon as possible.
- We investigate and work on a fix.
- For accepted reports, we coordinate disclosure and credit (unless you prefer anonymity).

Thank you for helping keep gfireui-backend and its users safe.
