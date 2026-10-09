# Service contracts (protobuf, gRPC)

Versioned contracts at bounded-context boundaries ([ADR-0007](../../docs/adr/0007-grpc-contracts-now-split-later.md)).

```text
api/proto/timesheetbot/
├── allocation/v1/policy.proto    PolicyService.Allocate (first: used by allocation-agent)
└── activity/v1/source.proto      ActivitySourceService.Collect
```

Rules: one package per context and version (`timesheetbot.<context>.v1`), the protobuf style guide, `buf lint` and `buf breaking` in CI, and no breaking changes inside `v1`. Generated Go code goes to the service that uses it. The files are written by `technical-architect` in the Phase 3 plan.
