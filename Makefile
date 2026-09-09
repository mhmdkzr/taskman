SHELL := /usr/bin/env bash

GOBIN ?= $(HOME)/go/bin
STATICCHECK := $(GOBIN)/staticcheck
GOVULNCHECK := $(GOBIN)/govulncheck
GO ?= go
GOLANGCI_LINT ?= golangci-lint
PKG_PATTERNS := ./...
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.worktrees/*')

.PHONY: lint golangci-lint tools fmt vet staticcheck govulncheck build install run test

lint: vet staticcheck golangci-lint govulncheck

# ── Backend ──────────────────────────────────────────────────────────────────

golangci-lint:
	@$(GOLANGCI_LINT) run $(PKG_PATTERNS)

tools:
	@command -v go >/dev/null
	@if [ ! -x "$(STATICCHECK)" ]; then \
		echo "missing required tool: $(STATICCHECK)"; \
		exit 1; \
	fi
	@if [ ! -x "$(GOVULNCHECK)" ]; then \
		echo "missing required tool: $(GOVULNCHECK)"; \
		exit 1; \
	fi

fmt:
	@gofmt -w $(GO_FILES)
	@goimports -w $(GO_FILES)

vet:
	@$(GO) vet $(PKG_PATTERNS)

staticcheck:
	@$(STATICCHECK) $(PKG_PATTERNS)

govulncheck:
	@$(GOVULNCHECK) $(PKG_PATTERNS)

build:
	@$(GO) build -o /dev/null .

install:
	@$(GO) install .

run:
	@$(GO) run .

test:
	@$(GO) test $(PKG_PATTERNS)
