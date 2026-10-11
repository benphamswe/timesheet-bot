.PHONY: build run test test-int test-race cover fuzz vuln lint lint-version fix arch ci

# Tool versions, the single source for local runs and CI. CI reads GOLANGCI_LINT_VERSION with
# `make -s print-GOLANGCI_LINT_VERSION` and runs govulncheck through `make vuln`.
GOLANGCI_LINT_VERSION := v2.14.0
GOVULNCHECK_VERSION   := v1.8.0
FUZZTIME              ?= 20s

build:
	CGO_ENABLED=0 go build -o bin/timesheet ./cmd/timesheet

# Go does not read .env files itself. LOAD_DOTENV exports this service's own .env
# (services/timesheet-core/.env, if present; gitignored, written by the owner only; ADR-0013) into
# the recipe's shell, then restores every variable already exported in your shell (~/.zshrc). So
# a token set only in .env is used, the shell wins when both set it, and an empty KEY= in .env
# never clears a shell token. It prints nothing and never echoes a value. Make runs recipes in
# this folder, also when called from the root (`make run` fans out with -C), so ./.env resolves
# here. The ./ matters: `. .env` without a slash would search PATH in a POSIX shell.
DOTENV      := ./.env
LOAD_DOTENV  = set -a; if [ -f $(DOTENV) ]; then shell_env=$$(export -p); . $(DOTENV); eval "$$shell_env"; fi; set +a;

# Runs the CLI with go run, e.g. `make -C services/timesheet-core run ARGS="plan --dry-run"`.
run:
	@$(LOAD_DOTENV) go run ./cmd/timesheet $(ARGS)

# Every test lives under test/ (ADR-0009); ./... covers it.
test:
	go test ./...

# Requires a running Postgres, e.g. `docker compose up -d db`. TEST_DATABASE_URL comes from the
# shell, then this service's .env, then the local compose default.
test-int:
	$(LOAD_DOTENV) TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgres://timesheet:timesheet@localhost:5432/timesheet?sslmode=disable} go test -tags=integration ./test/integration/...

test-race:
	go test -race ./...

# Domain and application packages must stay at or above 85% (qa-engineer). The tests live in
# test/, so -coverpkg is required: without it each test package only covers itself.
cover:
	go test -coverpkg=./internal/...,./cmd/... -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

# Runs every Fuzz target for FUZZTIME each.
fuzz:
	@for pkg in $$(go list ./test/...); do \
	  for fn in $$(go test -list '^Fuzz' $$pkg 2>/dev/null | grep '^Fuzz'); do \
	    echo "fuzz $$pkg $$fn"; go test -run='^$$' -fuzz="^$$fn$$" -fuzztime=$(FUZZTIME) $$pkg || exit 1; \
	  done; \
	done

# Runs the pinned govulncheck, so no local install is needed and the version cannot drift.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

arch:
	go test ./test/architecture/...

# Fails when the installed golangci-lint is not GOLANGCI_LINT_VERSION.
lint-version:
	@v=$$(golangci-lint version --short 2>/dev/null); \
	if [ "$${v#v}" != "$(GOLANGCI_LINT_VERSION:v%=%)" ]; then \
	  echo "golangci-lint '$$v' found, $(GOLANGCI_LINT_VERSION) required (GOLANGCI_LINT_VERSION)" >&2; exit 1; \
	fi

lint: lint-version
	golangci-lint run ./...

fix: lint-version
	golangci-lint run --fix ./...

# Prints a variable, e.g. `make -s print-GOLANGCI_LINT_VERSION`. CI uses it to read tool versions.
print-%:
	@echo '$($*)'

# Everything CI runs, in the same order.
ci: lint test test-race vuln build
