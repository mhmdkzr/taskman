SHELL := /usr/bin/env bash

GOBIN ?= $(HOME)/go/bin
STATICCHECK := $(GOBIN)/staticcheck
GOVULNCHECK := $(GOBIN)/govulncheck
WORKFLOWCHECK := $(GOBIN)/workflowcheck
GO ?= go
GOLANGCI_LINT ?= golangci-lint
DENO ?= deno
SRC_DIRS := cmd internal pkg
PKG_PATTERNS := $(addprefix ./, $(addsuffix /..., $(SRC_DIRS)))

FRONTEND := frontend

.PHONY: lint golangci-lint workflowcheck tools fmt vet staticcheck govulncheck test test-db test-e2e test-all
.PHONY: fe-dev fe-build fe-preview fe-check fe-lint fe-format fe-test fe-test-unit fe-test-e2e
.PHONY: fe-db-push fe-db-generate fe-db-migrate fe-db-studio fe-auth-schema

lint: vet workflowcheck staticcheck golangci-lint govulncheck

# ── Backend ──────────────────────────────────────────────────────────────────

golangci-lint:
	@$(GOLANGCI_LINT) run $(PKG_PATTERNS)

workflowcheck:
	@"$(WORKFLOWCHECK)" $(PKG_PATTERNS)

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
	@if [ ! -x "$(WORKFLOWCHECK)" ]; then \
		echo "missing required tool: $(WORKFLOWCHECK)"; \
		echo "install with: go install go.temporal.io/sdk/contrib/tools/workflowcheck@latest"; \
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

test:
	@$(GO) test $(PKG_PATTERNS)

test-db:
	@RUN_DB_TESTS=1 $(GO) test -count=1 -run '^TestDB' $(PKG_PATTERNS)

test-e2e:
	@RUN_E2E_TESTS=1 $(GO) test -count=1 $(PKG_PATTERNS)

test-all:
	@RUN_DB_TESTS=1 RUN_E2E_TESTS=1 $(GO) test -count=1 $(PKG_PATTERNS)

# ── Frontend ──────────────────────────────────────────────────────────────────

fe-dev:
	@cd "$(FRONTEND)" && $(DENO) task dev

fe-build:
	@cd "$(FRONTEND)" && $(DENO) task build

fe-preview:
	@cd "$(FRONTEND)" && $(DENO) task preview

fe-test-unit:
	@cd "$(FRONTEND)" && $(DENO) task test

# Requires the full Compose stack running (`docker compose up -d`) and
# provisioned (AUTH_ENABLED=true, scripts/provision-zitadel-bff.sh run) —
# see frontend/e2e/README.md.
fe-test-e2e:
	@cd "$(FRONTEND)" && $(DENO) task test:e2e

fe-test: fe-test-unit
