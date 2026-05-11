<p align="center">
  <img src="docs/assets/logo.svg" alt="npc - Nginx Proxy Configurator" width="760">
</p>

# npc

`npc` is the **Nginx Proxy Configurator**: a single Go binary for installing, configuring, testing, and managing Nginx reverse proxy sites on Linux.

It is built for administrators who want repeatable reverse proxy setup with safer defaults: backups before writes, `nginx -t` before reloads, explicit metadata, dry runs, and clear failure messages. This project started from a generated broad spec, so early releases should still be reviewed carefully before production rollout.

## Status

`v0.1.0` is the first MVP release. It includes the CLI structure, HTTP reverse proxy generation, manual certificate config, acme.sh scaffolding, backups, metadata, release builds, and tests. Some advanced flows are intentionally conservative and will mature over later releases.

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
