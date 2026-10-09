# timesheet-bot documentation

Code lives in `services/` (layout: [architecture/project-structure.md](architecture/project-structure.md)); gRPC contracts live in `api/proto/`.

| Folder | Owner | Contents |
|---|---|---|
| [backlog/](backlog/) | business-analyst | Epics, features, stories, tasks, sprints (mirrored to GitHub Issues by `/backlog-sync`) |
| [requirements/](requirements/) | business-analyst, solution-architect | FR and NFR, glossary, traceability |
| [architecture/](architecture/) | SA, TA, networking-engineer | Overview (C4), project structure, code map, bounded contexts, data model, testing, network, scalability |
| [features/](features/) | tech-writer | One doc per delivered feature (planned: first doc when FE-001 closes) |
| [learn/](learn/) | learning-coach | Beginner explanations, feature guides, Q&A from `/learn` |
| [adr/](adr/) | SA / TA | Architecture decision records |
| [adr/reviews/](adr/reviews/) | architecture-reviewer, security-engineer, networking-engineer, sre-engineer | Architecture board and production-readiness reviews of Proposed ADRs |
| [api/](api/) | technical-architect | External integrations and their verified contracts |
| [development/](development/) | technical-architect | First-time setup, conventions, SDLC agent workflow |
| [commands/](commands/) | tech-writer, every agent | Every command used on the project, grouped by purpose; add new commands here |
| [operations/](operations/) | sre-engineer, tech-writer | Deployment, configuration, runbooks. Planned (sre-engineer): SLOs, failure modes, postmortems |
| [security/](security/) | solution-architect | Threat model |
| [superpowers/specs/](superpowers/specs/) | - | Design specs (status is in each file) |
| [superpowers/plans/](superpowers/plans/) | technical-architect | Phase implementation plans |

Status legend used across docs: **[built]** implemented and tested, **[planned: Pn]** designed for phase n.

Current state: Phase 1 [built]. Phases 2-4 [planned].
