# Root Makefile for the monorepo (ADR-0008). It holds no language tooling: every target runs in
# each service that has its own Makefile, so `make ci` checks all services from the root.
SERVICES := $(patsubst %/Makefile,%,$(wildcard services/*/Makefile))

.DEFAULT_GOAL := ci
.PHONY: ci proto-lint collect

# Contracts first, then every service's own `ci` target.
ci: proto-lint
	@for s in $(SERVICES); do echo "== $$s: $@"; $(MAKE) --no-print-directory -C $$s $@ || exit 1; done

# Contracts: enabled once api/proto contains .proto files and buf.yaml. buf config stays in
# api/proto, so both buf inputs point there (buf.build/docs/reference/inputs).
proto-lint:
	@if find api/proto -name '*.proto' | grep -q .; then \
	  buf lint api/proto && buf breaking api/proto --against '.git#branch=main,subdir=api/proto'; \
	else echo "no proto files yet"; fi

# Slack activity collector (ADR-0011). Only services/allocation-agent has it, so it is not fanned
# out. Usage: make collect WEEK=2026-W41 [ARGS=--dry-run]; WEEK and ARGS reach the sub-make.
collect:
	@$(MAKE) --no-print-directory -C services/allocation-agent collect

# Any other target (test, lint, vuln, ...) runs in every service.
%:
	@for s in $(SERVICES); do echo "== $$s: $@"; $(MAKE) --no-print-directory -C $$s $@ || exit 1; done
