# Changelog

## 0.1.34 - 2026-06-24

- Refresh certificate metadata from existing npc-managed Nginx config files during site discovery.
- Added `npc certs issue <hostname>` for issuing certificates for existing managed/imported sites.
- `certs issue` supports HTTP-01 and DNS-01, including Cloudflare through saved acme.sh provider env files.
- Added a Web UI certificate issue form and automatic prefill for edit/certificate forms when selecting a site.

## 0.1.33 - 2026-06-24

- Replaced the web UI command console with normal AdminLTE form workflows.
- Added web forms for creating, editing, enabling, disabling, deleting, importing sites, and managing certificates.
- Added direct site table action buttons for enable, disable, and delete.
- Kept write confirmations on destructive or state-changing web UI actions.

## 0.1.32 - 2026-06-24

- Expanded `npc import` for manually created or third-party Nginx reverse proxy configs.
- Import now detects SSL certificate paths, HTTP/2, HTTPS redirects, WebSocket headers, access/error logs, body size, backend URL, and enabled symlink path.
- Added `npc import --path <file> --yes` and `npc import --yes --force` for targeted or metadata-refresh imports.
- Added `npc certs set <hostname>` to attach or update existing certificate/key paths and ACME metadata.
- Added `npc certs delete <hostname>` to remove certificate metadata, optionally remove acme.sh registration, and optionally delete certificate files.
- Added the new import and certificate operations to the AdminLTE web UI command catalog.

## 0.1.31 - 2026-06-24

- Added `npc webui` for an AdminLTE-based web interface.
- Added `--listen` to bind the web UI to a chosen interface and port.
- Added `npc webui unit`, `npc webui install-service`, and `npc webui uninstall-service` for persistent systemd operation.
- Added a dark-mode AdminLTE dashboard with site inventory, status cards, API endpoints, and an operations console.
- The operations console can run npc commands through argument lists without invoking a shell; write actions require explicit confirmation.

## 0.1.30 - 2026-06-14

- Added keyboard navigation to the dependency-light terminal UI.
- Menus now support Up/Down, Enter, and direct number selection in an Ubuntu-installer-style flow.
- Kept the previous numeric prompt fallback for terminals where raw keyboard input is unavailable.
- Kept Bubble Tea and heavy TUI dependencies out of the build.

## 0.1.29 - 2026-06-13

- Scan `/etc/nginx/sites-available/*.conf` for configs with the npc managed header.
- Treat discovered header configs as managed sites for `list`, `show`, `edit`, `delete`, `certs`, `check`, `maintenance`, backup, health output, and the terminal UI.
- Keep discovery read-only during listing and inspection; metadata is only persisted after an explicit write action.
- Prevent `create` from silently overwriting an existing npc-managed config unless `--force` is used.

## 0.1.28 - 2026-06-10

- Reverted the Bubble Tea TUI and restored the previous dependency-light terminal UI.
- Removed Bubble Tea, Lip Gloss, and related transitive dependencies from the build.
- Kept the large-site inventory features from `0.1.25`, including search, metadata, filters, health output, and scoped bulk actions.

## 0.1.25 - 2026-06-08

- Added site metadata for `alias`, `group`, `tags`, and `archived`.
- Added filtered and sortable `npc list` output with compact and `--wide` modes.
- Added `npc search <query>` across hostname, alias, group, tags, profile, backend, ACME method, and DNS provider.
- Added `npc set <site> --alias --group --tags` for metadata management.
- Added `npc archive` and `npc unarchive`; archived sites are hidden from normal lists.
- Added scoped `npc status`, scoped `npc backup`, `npc check --all`, and conservative bulk `npc disable --group/--tag --yes`.
- Added `npc certs renew --expiring [--days N]`.
- Expanded `npc health --only-problems` and the TUI site selector with search and problem badges.
- Bumped the config schema to version 2 and documented all new management arguments in the README.

## 0.1.24 - 2026-06-01

- Removed the unwanted alternative CA from supported and documented ACME CA choices.
- Reapply the selected acme.sh default CA before each certificate issuance; Let's Encrypt remains the default.
- Updated CLI help and README examples to avoid suggesting the unwanted CA.

## 0.1.23 - 2026-06-01

- Made Let's Encrypt the explicit default ACME CA across create, quick-create, and UI flows.
- Added `--acme-ca` for per-site CA selection.
- Added `npc acme default-ca [letsencrypt|buypass]` to set acme.sh's default CA manually.
- Stored the selected ACME CA in site metadata.

## 0.1.22 - 2026-06-01

- Fixed Cloudflare DNS-01 handling for acme.sh.
- Accept `CF_Token` plus `CF_Zone_ID` or `CF_Account_ID`; legacy `CF_Key` plus `CF_Email` remains supported.
- Parse shell-style env files with optional `export` and quoted values.
- Use Let's Encrypt explicitly for ACME issuance to avoid third-party account prompts in DNS-01 flows.

## 0.1.21 - 2026-06-01

- Prefer saved Cloudflare DNS-01 credentials automatically when `/etc/npc/secrets/cloudflare.env` is present and secure.
- Make Cloudflare DNS-01 the default ACME method in interactive create and Docker expose flows when credentials are available.
- Keep ACME account email optional when using saved Cloudflare credentials.

## 0.1.20 - 2026-06-01

- Improved the terminal UI styling with a stronger header, dashboard panels, action numbering, and clearer runtime/update sections.
- Added a terminal UI flow for Cloudflare DNS-01 setup.
- Added `npc acme cloudflare-setup` for writing Cloudflare DNS settings safely under `/etc/npc/secrets/cloudflare.env`.
- Added DNS-01 certificate issuance for acme.sh using provider env files without logging secrets.

## 0.1.19 - 2026-05-21

- Replaced the external Cobra dependency with a small local CLI layer tailored to npc commands.
- Removed the YAML dependency by adding a focused reader/writer for npc's own config format.
- Replaced embedded Go HTTP/TLS downloads with explicit `curl`/`wget` execution while keeping checksum verification.
- Switched certificate inspection to `openssl x509` to avoid embedding Go's full x509 parser.
- Added smaller deterministic release builds without binary compression or packers; local linux-amd64 release builds are now roughly half the previous binary size.

## 0.1.18 - 2026-05-21

- Added `npc diff <hostname>` to compare live configs, rendered configs, and saved revisions.
- Added `npc rollback <hostname>` with backup, revision restore, `nginx -t`, and safe reload behavior.
- Added `npc acme dns-setup <provider>` for DNS-01 provider env templates.
- Added `npc firewall suggest` for ufw, firewalld, and nftables guidance without changing rules.
- Added `npc migrate` for conservative config schema preparation.
- Added `npc monitor` and `npc health` for JSON, text, and Prometheus-style health output.
- Added shell completion artifacts to release workflows.

## 0.1.17 - 2026-05-21

- Expanded README documentation for proxy profiles, including choosing guidance, exact behavior, and command examples.
- Added a README section with useful next production features.

## 0.1.16 - 2026-05-21

- Added automatic GitHub Release update checks for CLI commands.
- Added `--no-upgrade` to skip the automatic update check for scripts and parsers.
- Added update status, latest version display, changelog display, and an upgrade action to the terminal UI.

## 0.1.15 - 2026-05-21

- Added terminal UI actions for editing managed sites.
- Added terminal UI actions for deleting managed sites with explicit choices for backups, config files, metadata, and certificate files.
- Documented the UI edit/delete lifecycle in the README.

## 0.1.14 - 2026-05-20

- Added `npc inspect <hostname>` for focused site runtime diagnostics, including symlink state, Nginx service state, certificate summary, and DNS comparison.
- Added `npc repair <hostname>` to re-render managed configs from metadata with revision capture, backup, `nginx -t`, and safe reload behavior.
- Expanded `npc certs` with certificate issuer and expiry parsing from PEM files.
- Updated ACME renew commands to use npc's acme.sh path discovery and added clearer diagnostics for common ACME failures.
- Added `npc backup list` and `npc backup restore <id-or-path>`.
- Improved `npc import` so manual Nginx sites can be reviewed and explicitly adopted into npc metadata.
- Added per-site config revisions under `/etc/npc/state/sites/<hostname>/revisions/<timestamp>/`.
- Strengthened proxy profiles for WebSocket, upload, streaming, API, WordPress, Nextcloud, Grafana, Node.js, and media workloads.

## 0.1.13 - 2026-05-11

- Added regression coverage for required proxy forwarding headers.
- Documented redirect-loop troubleshooting and the exact proxy headers generated by npc.

## 0.1.12 - 2026-05-11

- Added DNS preflight checks before acme.sh HTTP-01 issuance.
- Added public server IP detection and hostname A/AAAA comparison.
- Added network exit code handling for DNS/IP mismatch failures.
- Documented that HTTP-01 issuance requires DNS to point to the server.

## 0.1.11 - 2026-05-11

- Fixed acme.sh installer invocation for the official `get.acme.sh` script.
- Changed installer arguments from unsupported `--install`/`-m` flags to `email=<address>`.
- Deduplicated acme.sh search paths in installation error output.

## 0.1.10 - 2026-05-11

- Fixed acme.sh installer invocation to use the correct email argument.
- Added post-install validation that finds acme.sh outside of `PATH`.
- Added more acme.sh search paths, including `/root/.acme.sh/acme.sh`, user home paths, and common system paths.
- Improved acme.sh installation errors so certificate issuance cannot continue after a failed or incomplete install.

## 0.1.9 - 2026-05-11

- Added Nginx service active check before write/reload flows.
- Added automatic `systemctl start nginx` attempt when Nginx is installed but inactive.
- Added Nginx service active check before HTTP-01 challenge reload.
- Documented the service-active check in the README.

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
