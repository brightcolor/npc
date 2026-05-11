BINARY := npc
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
REPO_OWNER ?= example
REPO_NAME ?= npc
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.repoOwner=$(REPO_OWNER) -X main.repoName=$(REPO_NAME)

.PHONY: build test clean release install-local

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY) npc-linux-amd64 npc-linux-arm64 SHA256SUMS

release: clean
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o npc-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o npc-linux-arm64 .
	sha256sum npc-linux-amd64 npc-linux-arm64 > SHA256SUMS

install-local: build
	sudo ./$(BINARY) --install
