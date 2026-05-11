# Changelog

## 0.1.8 - 2026-05-11

- Changed `npc upgrade` to skip downloads and replacement when the installed version already matches the target release.
- Added clear `npc is already up to date` output.
- Documented no-op upgrade behavior.

## 0.1.7 - 2026-05-11

- Added startup checks in the terminal UI for Nginx and `acme.sh`.
- Added interactive install prompts for missing Nginx and `acme.sh` when launching `npc`.
- Added quick-create mode: `sudo npc <hostname> <port>`.
- Quick-create defaults to `http://127.0.0.1:<port>`, HTTPS via acme.sh HTTP-01, HTTPS redirect, WebSocket, HTTP/2, standard security headers, and per-site logs.
- Added staged acme.sh HTTP-01 issuance before writing the final HTTPS Nginx config.
- Documented startup dependency checks in the README.

## 0.1.6 - 2026-05-11

- Added README screenshots for the terminal dashboard and reverse proxy review screen.
- Improved terminal UI styling with stronger control-plane status, safer empty states, and richer review details.
- Suppressed Cobra usage output for runtime errors so operational failures stay readable.
- Improved create-flow error messages for failed Nginx and acme.sh installation attempts.
- Improved `npc upgrade` output to report the previous and target versions.

## 0.1.5 - 2026-05-11

- Refined the terminal UI with a stronger header, status badges, action cards, and clearer section styling.
- Improved the Docker discovery screen and reverse proxy review screen.
- Updated README wording to describe the polished terminal UI behavior.

## 0.1.4 - 2026-05-11

- Improved `npc list` empty-state output when no sites are managed yet.
- Improved the terminal UI with a dashboard for Nginx, Docker, and managed-site status.
- Added a managed-site list action inside the terminal UI.
- Improved terminal UI panels, prompts, and Docker discovery copy.
- Documented why `npc list` can be empty and how the UI reports empty states.

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
