# System overview

Status legend: **[built]**, **[planned: Pn]**. Decisions: [ADRs](../adr/). Contexts: [bounded-contexts.md](bounded-contexts.md). Tests: [testing.md](testing.md). Network: [network.md](network.md). Growth: [scalability.md](scalability.md).

## Context (C4 level 1)

```mermaid
flowchart LR
    owner([Owner])
    bot[timesheet-bot]
    slack[(Slack)]
    jira[(Jira Cloud)]
    tempo[(Tempo Cloud)]
    claude[(Anthropic API)]
    gh[(GitHub: CI + Issues mirror)]

    owner -- CLI now, Slack DM [P2] --> bot
    bot -- search.messages [built], DM + Socket Mode [P2] --> slack
    bot -- issue lookup [built] --> jira
    bot -- POST /4/worklogs [built] --> tempo
    bot -. allocation agent [P3] .-> claude
    owner -- push / review --> gh
```

## Containers (C4 level 2)

```mermaid
flowchart TB
    subgraph laptop[Owner laptop: Docker on WSL]
        app[timesheet container<br/>Go 1.26 static binary, distroless nonroot<br/>CLI now, `serve` daemon P2]
        db[(Postgres 17<br/>timesheets, entries, outbox)]
        ws[/agent/ volume: skills, memory P3-P4/]
        agent[allocation-agent container<br/>planned P3, ADR-0007<br/>Python, LangGraph, serves PolicyService over gRPC<br/>no published port]
    end
    app -- pgx, compose network --> db
    app -.-> ws
    app -- HTTPS via httpx.RetryTransport --> ext[Slack / Jira / Tempo]
    app -. gRPC .-> agent
    agent -.-> Anthropic[Anthropic API]
```

Under ADR-0007 the Anthropic egress belongs to the allocation-agent container, not the Go core (planned P3). See [ADR-0007](../adr/0007-grpc-contracts-now-split-later.md) and [network.md](network.md).

## Components (C4 level 3)

```mermaid
flowchart LR
    subgraph interfaces
        cli[cli] ; sched[scheduler P2] ; bot[slackbot P2] ; relay[outbox relay P2]
    end
    subgraph timesheet[Timesheet context]
        tapp[application: Propose, Transition, Submit] --> tdom[domain: Timesheet aggregate]
        tinf[infrastructure: postgres, jira, tempo, memory]
    end
    subgraph allocation[Allocation context]
        adom[domain: Policy, Proposal, RulePolicy] ; ainf[grpcpolicy: planned P3, ADR-0007]
    end
    subgraph activity[Activity context]
        actdom[domain: Source] ; actinf[slack collector]
    end
    kernel[[shared kernel]]
    wire[cmd/timesheet/wire.go<br/>composition root]

    cli --> tapp
    tapp --> adom
    tapp -. ports .-> tinf
    tapp -. ActivitySource port .-> actinf
    wire --> tinf & actinf & ainf & cli
    tdom & adom & actdom --> kernel
```

Dependency rule: arrows point inward to the domain and kernel. It is enforced by the architecture tests in `test/architecture` ([ADR-0001](../adr/0001-ddd-package-by-bounded-context.md); tests moved out of `internal/` by [ADR-0009](../adr/0009-tests-in-separate-tree.md)).

## Weekly flow

```mermaid
sequenceDiagram
    participant S as Scheduler [P2] / CLI
    participant P as ProposeHandler
    participant SL as Slack
    participant J as Jira
    participant DB as Postgres
    participant T as Tempo
    S->>P: ProposeTimesheet(week)
    P->>SL: search.messages from:me (user token)
    P->>J: resolve issue keys (drop unknown)
    P->>P: policies in order (agent [P3], rule) -> aggregate validation
    P->>DB: save timesheet + TimesheetProposed in one transaction (outbox)
    Note over S,DB: owner approves (CLI now, Slack button [P2])
    S->>T: SubmitHandler: POST /4/worklogs per entry
    S->>DB: save tempo_worklog_id after each entry
```

## Deployment and delivery

| Concern | How |
|---|---|
| Runtime | `docker compose` on the laptop; Postgres bound to 127.0.0.1 |
| Build | multi-stage `golang:1.26.9-alpine` to `distroless/static-debian12:nonroot`, tzdata and migrations embedded ([ADR-0005](../adr/0005-go-1-26-and-embedded-assets.md)) |
| CI | GitHub Actions `ci.yml`: lint, race and integration tests against Postgres, govulncheck, image build; actions pinned to commit SHAs |
| Dependencies | Dependabot weekly for Go modules, Actions and Docker; latest-stable policy (tech-selection P12) |
| Quality gates | `make ci` plus `make test-int cover` matches CI; architecture tests; the `/sdlc` agent pipeline |

## Quality attributes

See [non-functional requirements](../requirements/non-functional.md), [threat model](../security/threat-model.md), and SLOs (`docs/operations/slo.md`, planned, owned by sre-engineer).
