SHELL := /usr/bin/env bash

GOBIN ?= $(HOME)/go/bin
STATICCHECK := $(GOBIN)/staticcheck
GOVULNCHECK := $(GOBIN)/govulncheck
WORKFLOWCHECK := $(GOBIN)/workflowcheck
GO ?= go
GOLANGCI_LINT ?= golangci-lint
DENO ?= deno

FRONTEND := frontend

.PHONY: lint golangci-lint workflowcheck tools fmt vet staticcheck govulncheck test
.PHONY: fe-dev fe-build fe-preview fe-check fe-lint fe-format fe-test fe-test-unit fe-test-e2e
.PHONY: fe-db-push fe-db-generate fe-db-migrate fe-db-studio fe-auth-schema

lint: golangci-lint workflowcheck test fe-lint

# ── Backend ──────────────────────────────────────────────────────────────────

golangci-lint:
	@$(GOLANGCI_LINT) run ./...

workflowcheck:
	@if command -v "$(WORKFLOWCHECK)" >/dev/null 2>&1; then \
		$(GO) run "go.temporal.io/sdk/contrib/tools/workflowcheck" ./...; \
	fi

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
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt found unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	@$(GO) vet ./...

staticcheck:
	@$(STATICCHECK) ./...

govulncheck:
	@$(GOVULNCHECK) ./...

test:
	@$(GO) test ./...

# ── Frontend ──────────────────────────────────────────────────────────────────

fe-dev:
	@cd "$(FRONTEND)" && $(DENO) task dev

fe-build:
	@cd "$(FRONTEND)" && $(DENO) task build

fe-preview:
	@cd "$(FRONTEND)" && $(DENO) task preview
