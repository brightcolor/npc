<p align="center">
  <img src="docs/assets/logo.svg" alt="npc - Nginx Proxy Configurator" width="760">
</p>

# npc

`npc` is the **Nginx Proxy Configurator**: a single Go binary for installing, configuring, testing, and managing Nginx reverse proxy sites on Linux.

It is built for administrators who want repeatable reverse proxy setup with safer defaults: backups before writes, `nginx -t` before reloads, explicit metadata, dry runs, and clear failure messages. This project started from a generated broad spec, so early releases should still be reviewed carefully before production rollout.

## Screenshots

![npc terminal dashboard](docs/assets/screenshots/tui-dashboard.svg)

![npc reverse proxy review](docs/assets/screenshots/tui-review.svg)

## Status

`v0.1.x` is the first MVP line. It includes the CLI structure, terminal UI, Docker discovery, HTTP reverse proxy generation, manual certificate config, acme.sh scaffolding, backups, metadata, release builds, and tests. Some advanced flows are intentionally conservative and will mature over later releases.

## Install

```bash
curl -L -o npc https://github.com/brightcolor/npc/releases/latest/download/npc-linux-amd64
chmod +x npc
sudo ./npc --install
npc --version
```

For ARM64:

```bash
curl -L -o npc https://github.com/brightcolor/npc/releases/latest/download/npc-linux-arm64
chmod +x npc
sudo ./npc --install
```

`sudo ./npc --install` copies the running binary to `/usr/local/bin/npc`, backs up an existing binary as `/usr/local/bin/npc.bak.<timestamp>`, and sets executable permissions.

## Quick Start

Start the terminal UI:

```bash
npc
```

The UI can scan running Docker containers, list their available ports, and create a reverse proxy for the selected container. Published Docker ports are exposed through `127.0.0.1:<host-port>`. Container-only ports are offered with a warning because host Nginx must be able to reach the container name through networking.

The UI shows a dashboard with Nginx, Docker, and managed-site status before each action. It uses status badges, action cards, and a review screen before writing anything. If no sites exist yet, `npc list` and the UI show an empty-state message instead of returning a blank table.

At startup, the UI checks for Nginx and `acme.sh`. If either tool is missing, `npc` asks whether it should install it. Nginx is installed through `apt`; `acme.sh` is installed through the official installer. Installation requires root, so start the UI with `sudo npc` when you want npc to install missing dependencies.

`acme.sh` usually installs into `/root/.acme.sh/acme.sh` when `npc` runs as root. `npc` runs the official installer using `email=<address>` when an account email is provided, searches the install location directly, and does not require `acme.sh` to be available in `$PATH`.

Create a local reverse proxy interactively:

```bash
sudo npc create
```

Before writing a site, `npc create` checks whether Nginx is installed. If Nginx is missing, it asks before installing it through `apt`. When `--acme` is enabled, `npc` also checks for `acme.sh` and asks before installing it.

For unattended provisioning, combine `--non-interactive` with `--force`:

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --backend-scheme http \
  --non-interactive \
  --force
```

Create one non-interactively:

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --backend-scheme http \
  --non-interactive
```

Fast path with production defaults:

```bash
sudo npc app.example.com 3000
```

This shortcut means:

- public hostname: `app.example.com`
- backend: `http://127.0.0.1:3000`
- HTTPS enabled with acme.sh HTTP-01
- HTTP to HTTPS redirect enabled
- WebSocket headers enabled
- HTTP/2 enabled
- standard security headers enabled
- per-site access and error logs enabled
- no overwrite when the vHost already exists

The shortcut does not open the assistant. It stops only when validation fails, Nginx/acme.sh installation fails, certificate issuance fails, `nginx -t` fails, or the vHost already exists.

Preview without writing:

```bash
npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --backend-scheme http \
  --non-interactive \
  --dry-run
```

## How It Works

`npc` keeps the moving parts deliberately simple:

1. You describe a public hostname and a backend.
2. `npc` validates the input and checks local dependencies.
3. It checks whether the Nginx service is active and starts it when needed.
4. It renders an Nginx server config from an embedded template.
5. It writes the config to `/etc/nginx/sites-available/<hostname>.conf`.
6. It enables the site with a symlink in `/etc/nginx/sites-enabled/`.
7. It runs `nginx -t`.
8. It reloads Nginx only when the config test succeeds.
9. It stores site metadata in `/etc/npc/config.yaml`.

The generated Nginx config is a normal reverse proxy. Public traffic reaches Nginx on port 80 or 443, Nginx forwards the request to the backend service, and the backend receives standard proxy headers such as `X-Forwarded-For`, `X-Forwarded-Proto`, and `X-Real-IP`.

For acme.sh HTTP-01 sites, `npc` uses a staged flow. It first writes a temporary HTTP challenge-capable config, reloads Nginx after `nginx -t`, requests the certificate, installs the certificate under `/etc/npc/certs/<hostname>/`, then writes the final HTTPS config and reloads again after another config test.

Before HTTP-01 issuance, `npc` checks whether the hostname's A/AAAA records point to this server's public IP. If DNS does not match, certificate issuance is stopped because the ACME HTTP-01 challenge would fail from the public internet.

`npc` does not replace Nginx. It writes managed Nginx config files and leaves Nginx in charge of serving traffic.

## Docker Flow

When you run `npc` or `npc tui`, the terminal UI can scan Docker with:

```bash
docker ps --format '{{json .}}'
```

It reads container names, images, networks, and port mappings. If a container publishes a port like `0.0.0.0:8080->80/tcp`, `npc` proposes `127.0.0.1:8080` as the backend because that is reachable from host Nginx.

If a container only exposes an internal port like `80/tcp`, `npc` can still offer it, but it shows a warning. In that case Nginx on the host must be able to resolve and reach the container name, which usually requires deliberate Docker networking. `npc` does not modify Docker containers, Docker networks, or Compose files.

## Proxy Profiles

Profiles are presets for common reverse proxy behavior. They do not create a different kind of site; they adjust Nginx settings such as timeouts, WebSocket headers, and request-size expectations.

Use a profile as a starting point, then override individual flags when needed.

| Profile | Use case | What it changes |
| --- | --- | --- |
| `generic` | Standard web apps, dashboards, APIs | Balanced defaults, `60s` proxy read timeout, `100M` body size unless changed |
| `websocket` | Apps with WebSockets, realtime dashboards, Socket.IO | Longer read timeout and WebSocket-friendly behavior when used with `--websocket` |
| `upload` | File uploads, Nextcloud-like apps, large requests | Longer read timeout; usually pair with `--client-max-body-size 512M` or `1G` |
| `streaming` | SSE, long polling, streaming responses | Long read timeout for connections that intentionally stay open |
| `docker` | Backends discovered from Docker containers | Uses Docker container/port discovery and host-reachable backend defaults |
| `security-basic` | Small internal tools that need simple protection | Intended for stricter headers and Basic Auth flows as the security feature set expands |

Current template behavior is intentionally conservative: profiles mainly influence proxy timeout selection, while explicit flags such as `--websocket`, `--client-max-body-size`, `--security-headers`, `--access-log`, and `--error-log` control the visible Nginx directives.

Examples:

```bash
sudo npc create \
  --hostname ws.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 8080 \
  --profile websocket \
  --websocket \
  --non-interactive
```

```bash
sudo npc create \
  --hostname files.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 8080 \
  --profile upload \
  --client-max-body-size 1G \
  --non-interactive
```

## TLS and Certificates

For HTTP-only sites, Nginx listens on port 80 and proxies directly to the backend.

For HTTPS sites, `npc` can render a TLS server block with:

- `ssl_certificate`
- `ssl_certificate_key`
- TLS 1.2 and TLS 1.3
- HTTP/2 when `--http2` is set
- HTTP-to-HTTPS redirects when `--redirect-https` is set

There are two certificate modes:

- Existing certificate: pass `--cert-path` and `--key-path`.
- acme.sh: pass `--ssl --acme` and select `--acme-method http`, `dns`, `standalone`, or `tls-alpn`.

When ACME is enabled, `npc` checks whether `acme.sh` is installed and asks before installing it. DNS provider secrets must never be pasted into logs; keep them in `/etc/npc/secrets/<provider>.env` with mode `0600`.

## Upgrade Flow

`npc upgrade` updates the installed binary from GitHub Releases.

```bash
sudo npc upgrade
```

By default it uses the latest release and selects the asset for the current platform:

- `npc-linux-amd64`
- `npc-linux-arm64`

The upgrade flow downloads the binary and `SHA256SUMS`, verifies the checksum, backs up the current binary as `<target>.bak.<timestamp>`, writes the new binary, and replaces the old one atomically. If replacing the binary fails, `npc` tries to roll back to the backup. On success, it prints the source and target versions, for example `Upgraded npc from v0.1.5 to v0.1.6`.

If the installed version already matches the selected release, `npc upgrade` exits without downloading or replacing the binary.

Install a specific release:

```bash
sudo npc upgrade --version v0.1.3
```

When `npc` is installed at `/usr/local/bin/npc`, upgrade requires root because that path is system-owned.

## Examples

### Local App

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --backend-scheme http \
  --non-interactive
```

### Docker Backend

```bash
npc docker
npc
```

Inside the UI, choose **Expose a Docker container**, select a running container, select one of its ports, enter the public hostname, review the generated Nginx config, and confirm.

The direct non-interactive equivalent is:

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host container-name \
  --backend-port 8080 \
  --profile docker \
  --non-interactive
```

### WebSocket App

```bash
sudo npc create \
  --hostname ws.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 8080 \
  --websocket \
  --profile websocket \
  --non-interactive
```

### Upload Profile

```bash
sudo npc create \
  --hostname files.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 8080 \
  --profile upload \
  --client-max-body-size 1G \
  --non-interactive
```

### Existing TLS Certificate

```bash
sudo npc create \
  --hostname secure.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --ssl \
  --http2 \
  --redirect-https \
  --cert-path /etc/ssl/example/fullchain.pem \
  --key-path /etc/ssl/example/privkey.pem \
  --non-interactive
```

### acme.sh HTTP-01

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --ssl \
  --acme \
  --acme-method http \
  --email admin@example.com \
  --redirect-https \
  --non-interactive
```

### acme.sh DNS-01 with Cloudflare

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --ssl \
  --acme \
  --acme-method dns \
  --dns-provider cloudflare \
  --email admin@example.com \
  --redirect-https \
  --non-interactive
```

## Commands

```bash
npc
npc tui
npc list
npc status
npc show app.example.com
sudo npc edit app.example.com --backend-port 3001
sudo npc disable app.example.com
sudo npc enable app.example.com
sudo npc delete app.example.com --force
npc certs
npc doctor
sudo npc backup
npc restore
npc docker
```

`npc list` only shows sites that were created or imported into npc metadata. Existing manual Nginx configs are intentionally not listed until they are imported.

## Managed Files

```text
/etc/npc/config.yaml
/etc/npc/secrets/
/etc/npc/certs/
/etc/npc/backups/
/etc/npc/auth/
/etc/npc/templates/
/etc/npc/state/
/etc/npc/sites/
/etc/nginx/sites-available/<hostname>.conf
/etc/nginx/sites-enabled/<hostname>.conf
```

Every generated Nginx config starts with:

```nginx
# Managed by npc
# Do not edit manually unless you know what you are doing.
# Hostname: <hostname>
```

## Safety Model

- Read-only commands should work without root.
- Write commands require root.
- `npc create` checks for Nginx before writing and asks before installing it.
- `npc create --acme` checks for `acme.sh` before writing and asks before installing it.
- `--non-interactive` never prompts; missing dependencies fail cleanly unless `--force` is set.
- The Docker UI does not modify Docker containers or Compose files. It only uses container/port information to generate Nginx reverse proxy config.
- Existing manual Nginx configs are not overwritten by default.
- Reload and restart paths run `nginx -t` first.
- `--dry-run` shows planned files and rendered config.
- Backups are written under `/etc/npc/backups/<timestamp>/`.
- Secrets belong in `/etc/npc/secrets/<provider>.env` with mode `0600`.

## Build

```bash
make build
make test
make release
```

Release artifacts:

- `npc-linux-amd64`
- `npc-linux-arm64`
- `SHA256SUMS`

## Troubleshooting

```bash
npc doctor
npc test
systemctl status nginx
journalctl -u nginx
```

For HTTP-01, DNS must point at the host and port 80 must be reachable. For public HTTPS traffic, port 443 must be reachable. DNS-01 does not require inbound validation ports, but provider secrets must be protected.

## Uninstall

```bash
sudo npc uninstall --force
```

The current MVP removes the binary. Review `/etc/npc`, managed Nginx configs, certificates, backups, and Nginx itself before deleting them.
