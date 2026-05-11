# npc

`npc` is the Nginx Proxy Configurator: a Go-based single-binary CLI for installing, configuring, testing, and managing Nginx reverse proxy sites.

It is aimed at Linux administrators who want fast, repeatable reverse proxy setup without hand-writing every Nginx server block. The project was generated from a broad initial spec, so treat early releases as carefully reviewed admin tooling rather than magic.

## Why npc?

- Single Linux binary for `linux-amd64` and `linux-arm64`
- Safe defaults: backup before writes, `nginx -t` before reloads, no unmanaged config overwrite by default
- Clear metadata in `/etc/npc/config.yaml`
- Managed Nginx files are marked with `# Managed by npc`
- Supports dry runs for write-heavy flows
- Prepared for acme.sh, DNS providers, maintenance mode, Docker discovery, and self-updates

## Installation

```bash
curl -L -o npc https://github.com/<owner>/<repo>/releases/latest/download/npc-linux-amd64
chmod +x npc
sudo ./npc --install
npc --version
```

The installer copies the running binary to `/usr/local/bin/npc`, backs up an existing binary as `/usr/local/bin/npc.bak.<timestamp>`, sets executable permissions, and keeps the command available system-wide.

## Quick Start

Interactive mode:

```bash
sudo npc create
```

Non-interactive HTTP reverse proxy:

```bash
sudo npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --backend-scheme http \
  --non-interactive
```

Dry run:

```bash
npc create \
  --hostname app.example.com \
  --backend-host 127.0.0.1 \
  --backend-port 3000 \
  --backend-scheme http \
  --non-interactive \
  --dry-run
```

## Examples

Local app:

```bash
sudo npc create --hostname app.example.com --backend-host 127.0.0.1 --backend-port 3000 --backend-scheme http --non-interactive
```

Docker container:

```bash
npc docker
sudo npc create --hostname app.example.com --backend-host container-name --backend-port 8080 --profile docker --non-interactive
```

WebSocket app:

```bash
sudo npc create --hostname ws.example.com --backend-host 127.0.0.1 --backend-port 8080 --websocket --profile websocket --non-interactive
```

Upload profile:

```bash
sudo npc create --hostname files.example.com --backend-host 127.0.0.1 --backend-port 8080 --profile upload --client-max-body-size 1G --non-interactive
```

Existing certificate:

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

acme.sh HTTP-01 metadata and config preparation:

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

DNS-01 with Cloudflare metadata:

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

## Common Commands

```bash
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
```

## SSL with acme.sh

`npc` includes command construction and metadata for acme.sh flows. Early releases prepare certificate paths and Nginx config, and expose `npc certs renew` / `npc certs renew-all`. Full issuance orchestration should be reviewed per environment because DNS provider secrets and CA behavior vary.

Secrets must be stored under `/etc/npc/secrets/<provider>.env` with mode `0600`. Do not paste secrets into logs, issue reports, or shell history.

## Files

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

## Safety Notes

- Read-only commands do not require root.
- Write commands require root.
- Existing manual Nginx configs are not overwritten unless forced.
- Every reload path runs `nginx -t` first.
- Use `--dry-run` before production changes.
- Backups are stored under `/etc/npc/backups/<timestamp>/`.

## Troubleshooting

Run:

```bash
npc doctor
npc test
systemctl status nginx
journalctl -u nginx
```

Check DNS before HTTP-01 or TLS-ALPN-01 issuance. Port 80 must be reachable for HTTP-01 and port 443 for HTTPS traffic. DNS-01 does not require inbound validation ports.

## Update

```bash
sudo npc upgrade
```

Self-upgrade is scaffolded around GitHub Releases, platform artifacts, and SHA256 verification. Configure `repoOwner` and `repoName` at build time.

## Deinstallation

```bash
sudo npc uninstall --force
```

The MVP removes the binary only. Review `/etc/npc`, managed Nginx configs, certificates, backups, and Nginx itself manually before deleting them.
