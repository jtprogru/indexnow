# indexnow — developer entrypoints.
#
# These targets are the single source of truth for build / test / lint / docs
# commands. CI calls the same targets instead of repeating the flags inline, so
# `make ci` locally and the workflow run cannot drift apart.

BINARY_NAME  ?= indexnow
MAIN_PKG     ?= ./cmd/indexnow
DIST_DIR     ?= ./dist
SITE_DIR     ?= ./site
COVERPROFILE ?= cover.out

# Extra arguments for `make run`, e.g. `make run ARGS="submit --help"`.
ARGS ?=

GO     ?= go
GOFMT  ?= gofmt
PIP    ?= pip
MKDOCS ?= mkdocs

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

.DEFAULT_GOAL := help

.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

## --- Go ---------------------------------------------------------------------

.PHONY: run
run: ## Run via go run (ARGS="submit --help")
	$(GO) run $(MAIN_PKG) $(ARGS)

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: build
build: ## Build the binary into ./dist
	$(GO) mod download
	CGO_ENABLED=0 $(GO) build -o $(DIST_DIR)/$(BINARY_NAME) $(MAIN_PKG)

.PHONY: install
install: ## go install into GOBIN
	$(GO) install $(MAIN_PKG)

.PHONY: fmt
fmt: ## gofmt -s -w .
	$(GOFMT) -s -w .

.PHONY: vet
vet: ## go vet ./...
	$(GO) vet ./...

.PHONY: test
test: ## Unit tests with coverage (short mode)
	$(GO) test --short -coverprofile=$(COVERPROFILE) -v ./...

.PHONY: test-race
test-race: ## Tests under the race detector with coverage — what CI runs
	$(GO) test -race -v -coverprofile=$(COVERPROFILE) ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint -v run

.PHONY: ci
ci: lint test-race ## Everything CI runs (lint + race tests). Quick local pre-push check.

.PHONY: release-dry
release-dry: ## Goreleaser dry-run (no publish; binaries land in ./dist)
	goreleaser release --clean --snapshot --skip=publish,sign

## --- Docs -------------------------------------------------------------------

.PHONY: docs-install
docs-install: ## Install MkDocs + plugins into the current Python env
	$(PIP) install -r docs/requirements.txt

.PHONY: docs-serve
docs-serve: ## Serve the docs site locally at http://127.0.0.1:8000
	$(MKDOCS) serve

.PHONY: docs-build
docs-build: ## Build the docs site to ./site (no deploy)
	$(MKDOCS) build --strict

.PHONY: docs-deploy
docs-deploy: ## Build and publish the docs site to gh-pages
	$(MKDOCS) gh-deploy --force --no-history

## --- Housekeeping -----------------------------------------------------------

.PHONY: clean
clean: ## Remove build, coverage and docs artifacts
	rm -rf $(DIST_DIR) $(SITE_DIR) $(COVERPROFILE)
