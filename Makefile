SHELL := /usr/bin/env bash

GOBIN ?= $(HOME)/go/bin
STATICCHECK := $(GOBIN)/staticcheck
GOVULNCHECK := $(GOBIN)/govulncheck
GO ?= go
GOLANGCI_LINT ?= golangci-lint
TEMPL := $(GOBIN)/templ
SRC_DIRS := cmd internal pkg
PKG_PATTERNS := $(addprefix ./, $(addsuffix /..., $(SRC_DIRS)))

.PHONY: lint golangci-lint tools fmt vet staticcheck govulncheck generate build run test test-db test-e2e test-all

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
	@if [ ! -x "$(TEMPL)" ]; then \
		echo "missing required tool: $(TEMPL)"; \
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

generate:
	@$(GO) generate ./...

build: generate
	@$(GO) build -o /dev/null ./cmd/main

run: generate
	@$(GO) run ./cmd/main

test:
	@$(GO) test $(PKG_PATTERNS)

test-db:
	@$(GO) test -count=1 -run '^TestDB' $(PKG_PATTERNS)

test-e2e:
	@RUN_E2E_TESTS=1 $(GO) test -count=1 $(PKG_PATTERNS)

test-all:
	@RUN_E2E_TESTS=1 $(GO) test -count=1 $(PKG_PATTERNS)
