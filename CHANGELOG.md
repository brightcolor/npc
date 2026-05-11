# Changelog

## 0.1.3 - 2026-05-11

- Expanded README with a general explanation of how `npc` works.
- Documented Docker discovery and port selection behavior.
- Documented proxy profiles and when to use each one.
- Documented TLS/certificate modes and the relationship between Nginx and generated configs.
- Implemented `npc upgrade` with GitHub Release downloads, SHA256 verification, binary backup, atomic replacement, and rollback attempt on replacement failure.
- Added `npc upgrade --version <tag>` for installing a specific SemVer release.
- Fixed the Docker backend example formatting.

## 0.1.2 - 2026-05-11

- Added `npc tui` and made bare `npc` open the terminal UI.
- Added Docker container discovery in the UI.
- Added selectable Docker port exposure flow with generated reverse proxy preview.
- Improved `npc docker` with structured Docker parsing and JSON output support.
- Documented the Docker expose UI in the README.

## 0.1.1 - 2026-05-11

- Added dependency checks during `npc create`.
- Added interactive installation prompt for missing Nginx before writing site configs.
- Added interactive installation prompt for missing `acme.sh` when `--acme` is enabled.
- Documented dependency behavior and unattended `--non-interactive --force` provisioning.

## 0.1.0 - 2026-05-11

- Added Go module and Cobra CLI for `npc`.
- Added `--version`, `--install`, `create`, `list`, `status`, `show`, `edit`, `enable`, `disable`, `delete`, `test`, `reload`, `restart`, `certs`, `doctor`, `backup`, `restore`, `docker`, `maintenance`, `check`, `import`, `export`, `install-nginx`, and `uninstall`.
- Added embedded Nginx template rendering with HTTP, HTTPS, redirect, WebSocket, security header, log, and maintenance support.
- Added YAML metadata storage under `/etc/npc/config.yaml`.
- Added backup helpers, root checks, safe command execution wrappers, acme.sh command scaffolding, secure secret file helper, checksum verification helper, Makefile, and GitHub Actions release workflow.
- Added unit tests for validation, config read/write, Nginx rendering, ACME command building, checksum verification, and secret permissions.
- Added README branding, installation URLs for `brightcolor/npc`, and SVG logo.
