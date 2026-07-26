SHELL := /usr/bin/env bash

GOBIN ?= $(HOME)/go/bin
STATICCHECK := $(GOBIN)/staticcheck
GOVULNCHECK := $(GOBIN)/govulncheck
WORKFLOWCHECK := $(GOBIN)/workflowcheck
GO ?= go
GOLANGCI_LINT ?= golangci-lint
DENO ?= deno

BACKEND := backend
FRONTEND := frontend

.PHONY: lint golangci-lint workflowcheck tools fmt vet staticcheck govulncheck test
.PHONY: fe-dev fe-build fe-preview fe-check fe-lint fe-format fe-test fe-test-unit fe-test-e2e
.PHONY: fe-db-push fe-db-generate fe-db-migrate fe-db-studio fe-auth-schema

lint: golangci-lint workflowcheck test fe-lint

# ── Backend ──────────────────────────────────────────────────────────────────

golangci-lint:
	@$(GOLANGCI_LINT) run ./$(BACKEND)/...

workflowcheck:
	@if command -v "$(WORKFLOWCHECK)" >/dev/null 2>&1; then \
		$(GO) run -C "$(BACKEND)" "go.temporal.io/sdk/contrib/tools/workflowcheck" ./...; \
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
	@unformatted="$$(gofmt -l ./$(BACKEND))"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt found unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	@$(GO) vet -C "$(BACKEND)" ./...

staticcheck:
	@$(STATICCHECK) ./$(BACKEND)/...

govulncheck:
	@$(GOVULNCHECK) ./$(BACKEND)/...

test:
	@$(GO) test -C "$(BACKEND)" ./...

# ── Frontend ──────────────────────────────────────────────────────────────────

fe-dev:
	@cd "$(FRONTEND)" && $(DENO) run -A dev

fe-build:
	@cd "$(FRONTEND)" && $(DENO) run -A build

fe-preview:
	@cd "$(FRONTEND)" && $(DENO) run -A preview

fe-check:
	@cd "$(FRONTEND)" && $(DENO) run -A check

fe-lint:
	@cd "$(FRONTEND)" && $(DENO) run -A lint

fe-format:
	@cd "$(FRONTEND)" && $(DENO) run -A format

fe-test:
	@cd "$(FRONTEND)" && $(DENO) run -A test:unit -- --run && $(DENO) run -A test:e2e

fe-test-unit:
	@cd "$(FRONTEND)" && $(DENO) run -A test:unit

fe-test-e2e:
	@cd "$(FRONTEND)" && $(DENO) run -A test:e2e

fe-db-push:
	@cd "$(FRONTEND)" && $(DENO) run -A db:push

fe-db-generate:
	@cd "$(FRONTEND)" && $(DENO) run -A db:generate

fe-db-migrate:
	@cd "$(FRONTEND)" && $(DENO) run -A db:migrate

fe-db-studio:
	@cd "$(FRONTEND)" && $(DENO) run -A db:studio

fe-auth-schema:
	@cd "$(FRONTEND)" && $(DENO) run -A auth:schema
