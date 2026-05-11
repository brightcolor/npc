# Architecture

`npc` is organized around small internal packages:

- `cmd`: Cobra commands and CLI orchestration.
- `internal/config`: YAML metadata for managed sites.
- `internal/renderer`: embedded Nginx templates.
- `internal/nginx`: Nginx file and service operations.
- `internal/backup`: timestamped backup sets.
- `internal/acme`: acme.sh command construction and provider mapping.
- `internal/secrets`: secure secret file handling.
- `internal/updater`: checksum helpers for release updates.

Write commands require root, read-only commands should stay usable without root, and every path that reloads Nginx must run `nginx -t` first.
