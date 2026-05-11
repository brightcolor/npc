# Changelog

## 0.1.0-dev

- Added Go module and Cobra CLI for `npc`.
- Added `--version`, `--install`, `create`, `list`, `status`, `show`, `edit`, `enable`, `disable`, `delete`, `test`, `reload`, `restart`, `certs`, `doctor`, `backup`, `restore`, `docker`, `maintenance`, `check`, `import`, `export`, `install-nginx`, and `uninstall`.
- Added embedded Nginx template rendering with HTTP, HTTPS, redirect, WebSocket, security header, log, and maintenance support.
- Added YAML metadata storage under `/etc/npc/config.yaml`.
- Added backup helpers, root checks, safe command execution wrappers, acme.sh command scaffolding, secure secret file helper, checksum verification helper, Makefile, and GitHub Actions release workflow.
- Added unit tests for validation, config read/write, Nginx rendering, ACME command building, checksum verification, and secret permissions.
