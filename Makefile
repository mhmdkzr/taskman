SHELL := /usr/bin/env bash

GOBIN ?= $(HOME)/go/bin
STATICCHECK := $(GOBIN)/staticcheck
GOVULNCHECK := $(GOBIN)/govulncheck
GO ?= go
GOLANGCI_LINT ?= golangci-lint
SRC_DIRS := cmd internal
PKG_PATTERNS := $(addprefix ./, $(addsuffix /..., $(SRC_DIRS)))

.PHONY: lint golangci-lint tools fmt vet staticcheck govulncheck build run test

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
	@gofmt -w $(SRC_DIRS)
	@goimports -w $(SRC_DIRS)
	@files=$$(git ls-files '*.md' | grep -Ev '(^|/)vendor/|^scripts/mdjsonfmt/testdata/'); \
	if [ -n "$$files" ]; then scripts/mdjsonfmt/mdjsonfmt.sh $$files; fi

vet:
	@$(GO) vet $(PKG_PATTERNS)

staticcheck:
	@$(STATICCHECK) $(PKG_PATTERNS)

govulncheck:
	@$(GOVULNCHECK) $(PKG_PATTERNS)

build:
	@$(GO) build -o /dev/null ./cmd/main

run:
	@$(GO) run ./cmd/main

test:
	@$(GO) test $(PKG_PATTERNS)
