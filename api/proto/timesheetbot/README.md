# timesheetbot protobuf packages

This directory holds the `.proto` files, one package per bounded context and version: `timesheetbot/<context>/v1/` with package `timesheetbot.<context>.v1`. It is empty until `technical-architect` writes the first contracts in the Phase 3 plan `[planned: P3]`.

Layout, rules and the list of planned files: [`api/proto/README.md`](../README.md). The `buf.yaml` lives in `api/proto/`; `make proto-lint` from the repository root runs `buf lint` and `buf breaking` once `.proto` files exist.
