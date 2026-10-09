# Traceability: acceptance criteria to tests

Maintained by `business-analyst` (mode `acceptance`). One row per acceptance criterion.

The test location is the folder under `services/timesheet-core/` that holds the test. Every Go test lives in `test/`, black-box, outside the production packages ([ADR-0009](../adr/0009-tests-in-separate-tree.md)); the folder for each package follows the mapping in [testing](../architecture/testing.md#layout-by-test-type). The test file carries the AC id in the test name or in a comment on the test.

FE-001 rows (AC-001 to AC-005) name the real tests; results are from 2026-10-09. FE-005 rows (AC-006 to AC-029) were added on 2026-10-09 when the stories were written. Their test is `planned` and is named at acceptance; the location is the expected one.

| AC id | Test | Test location | Result |
|---|---|---|---|
| AC-001.1 | `TestCollectPagesAndLocalDays` | test/unit/adapters/slack | pass |
| AC-001.2 | `TestExtractIssueKeys` | test/unit/kernel | pass |
| AC-002.1 | `TestRulePolicyOneEntryPerKeyPerDay` | test/unit/allocation/domain | pass |
| AC-002.2 | `TestRulePolicyChannelDefault` | test/unit/allocation/domain | pass |
| AC-002.3 | `TestRulePolicyScalesOverCapDay` | test/unit/allocation/domain | skipped (owner task, `TODO(owner)`, US-002-T2) |
| AC-003.1 | `TestProposeWithRulePolicy` | test/unit/timesheet/application | pass |
| AC-003.2 | `TestProposeAlreadyProposedAndForce` | test/unit/timesheet/application | pass |
| AC-003.3 | `TestProposeFallsBackWhenProposalInvalid` | test/unit/timesheet/application | pass |
| AC-003.4 | `TestRoundTripConflictAndOutbox` | test/integration/postgres | pass (`make test-int`) |
| AC-004.1 | `TestIllegalTransitions` | test/unit/timesheet/domain | pass |
| AC-004.2 | `TestRejectRecordsReason` | test/unit/timesheet/application | pass |
| AC-005.1 | `TestSubmitStaggersStartTimes` | test/unit/timesheet/application | pass |
| AC-005.2 | `TestSubmitPartialFailureThenRetry` | test/unit/timesheet/application | pass |
| AC-005.3 | `TestCreateDoesNotResendWorklogOn5xx` | test/unit/adapters/tempo | skipped (BUG H2; un-skip with US-005-T2) |
| AC-006.1 | planned | test/unit/timesheet/application, test/integration/postgres | not run |
| AC-006.2 | planned | test/unit/timesheet/application, test/integration/postgres | not run |
| AC-006.3 | planned | test/unit/timesheet/application, test/integration/postgres | not run |
| AC-007.1 | planned | test/unit/timesheet/application, test/unit/adapters/tempo | not run |
| AC-007.2 | planned | test/unit/timesheet/application, test/unit/adapters/tempo | not run |
| AC-007.3 | planned | test/unit/timesheet/application, test/unit/adapters/tempo | not run |
| AC-007.4 | planned | test/unit/timesheet/application, test/unit/adapters/tempo | not run |
| AC-008.1 | planned | test/unit/timesheet/domain | not run |
| AC-008.2 | planned | test/unit/timesheet/domain | not run |
| AC-008.3 | planned | test/unit/timesheet/domain | not run |
| AC-008.4 | planned | test/unit/timesheet/domain | not run |
| AC-009.1 | planned | test/integration/postgres, test/unit/adapters/memory | not run |
| AC-009.2 | planned | test/integration/postgres, test/unit/adapters/memory | not run |
| AC-010.1 | planned | test/contract (RunRepositoryContract), test/unit/adapters/memory, test/integration/postgres | not run |
| AC-010.2 | planned | test/contract (RunRepositoryContract), test/unit/adapters/memory, test/integration/postgres | not run |
| AC-010.3 | planned | test/contract (RunRepositoryContract), test/unit/adapters/memory, test/integration/postgres | not run |
| AC-010.4 | planned | test/contract (RunRepositoryContract), test/unit/adapters/memory, test/integration/postgres | not run |
| AC-011.1 | planned | test/unit/timesheet/domain, test/integration/postgres | not run |
| AC-011.2 | planned | test/unit/timesheet/domain, test/integration/postgres | not run |
| AC-011.3 | planned | test/unit/timesheet/domain, test/integration/postgres | not run |
| AC-012.1 | planned | test/architecture | not run |
| AC-012.2 | planned | test/architecture | not run |
| AC-012.3 | planned | test/architecture | not run |
| AC-012.4 | planned | test/architecture | not run |
| AC-012.5 | planned | test/architecture | not run |
| AC-013.1 | planned | test/unit/interfaces/cli, test/unit/timesheet/application, test/architecture | not run |
| AC-013.2 | planned | test/unit/interfaces/cli, test/unit/timesheet/application, test/architecture | not run |
| AC-013.3 | planned | test/unit/interfaces/cli, test/unit/timesheet/application, test/architecture | not run |
| AC-013.4 | planned | test/unit/interfaces/cli, test/unit/timesheet/application, test/architecture | not run |
| AC-014.1 | planned | test/architecture | not run |
| AC-014.2 | planned | test/architecture | not run |
| AC-015.1 | `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0` (`make lint`) | make lint (golangci-lint) | pass (0 issues, 2026-10-09) |
| AC-015.2 | planned | make lint (golangci-lint) | not run |
| AC-015.3 | planned | make lint (golangci-lint) | not run |
| AC-016.1 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-016.2 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-016.3 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-016.4 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-016.5 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-016.6 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-016.7 | planned | test/unit/timesheet/domain, test/unit/adapters/slacksearch, test/integration/platform/db, test/architecture | not run |
| AC-017.1 | planned | test/unit/httpx, test/unit/timesheet/application, test/unit/adapters/jira, test/unit/adapters/tempo | not run |
| AC-017.2 | planned | test/unit/httpx, test/unit/timesheet/application, test/unit/adapters/jira, test/unit/adapters/tempo | not run |
| AC-017.3 | planned | test/unit/httpx, test/unit/timesheet/application, test/unit/adapters/jira, test/unit/adapters/tempo | not run |
| AC-017.4 | planned | test/unit/httpx, test/unit/timesheet/application, test/unit/adapters/jira, test/unit/adapters/tempo | not run |
| AC-017.5 | `TestRetryReplaysNoBodyRequest` | test/unit/httpx | skipped (BUG NoBody; un-skip with US-017-T4) |
| AC-018.1 | planned | test/integration/postgres | not run |
| AC-018.2 | planned | test/integration/postgres | not run |
| AC-018.3 | planned | test/integration/postgres | not run |
| AC-019.1 | planned | test/e2e, test/unit/interfaces/cli, test/unit/config | not run |
| AC-019.2 | planned | test/e2e, test/unit/interfaces/cli, test/unit/config | not run |
| AC-019.3 | planned | test/e2e, test/unit/interfaces/cli, test/unit/config | not run |
| AC-019.4 | planned | test/e2e, test/unit/interfaces/cli, test/unit/config | not run |
| AC-020.1 | planned | test/e2e, test/unit/interfaces/cli, test/unit/timesheet/application | not run |
| AC-020.2 | planned | test/e2e, test/unit/interfaces/cli, test/unit/timesheet/application | not run |
| AC-020.3 | planned | test/e2e, test/unit/interfaces/cli, test/unit/timesheet/application | not run |
| AC-021.1 | planned | CI check (Dockerfile, docker-compose.yml, .github/dependabot.yml) | not run |
| AC-021.2 | planned | CI check (Dockerfile, docker-compose.yml, .github/dependabot.yml) | not run |
| AC-021.3 | planned | CI check (Dockerfile, docker-compose.yml, .github/dependabot.yml) | not run |
| AC-021.4 | planned | CI check (Dockerfile, docker-compose.yml, .github/dependabot.yml) | not run |
| AC-022.1 | planned | test/e2e | not run |
| AC-022.2 | planned | test/e2e | not run |
| AC-022.3 | planned | test/e2e | not run |
| AC-022.4 | planned | test/e2e | not run |
| AC-023.1 | planned | test/unit/adapters/tempo, test/unit/adapters/jira, test/unit/adapters/slacksearch | not run |
| AC-023.2 | planned | test/unit/adapters/tempo, test/unit/adapters/jira, test/unit/adapters/slacksearch | not run |
| AC-023.3 | planned | test/unit/adapters/tempo, test/unit/adapters/jira, test/unit/adapters/slacksearch | not run |
| AC-024.1 | planned | Docker smoke test (CI) | not run |
| AC-024.2 | planned | Docker smoke test (CI) | not run |
| AC-024.3 | planned | Docker smoke test (CI) | not run |
| AC-025.1 | planned | test/unit/adapters/*, test/unit/config, test/e2e | not run |
| AC-025.2 | planned | test/unit/adapters/*, test/unit/config, test/e2e | not run |
| AC-025.3 | planned | test/unit/adapters/*, test/unit/config, test/e2e | not run |
| AC-026.1 | planned | test/unit/kernel, test/unit/adapters/tempo, test/unit/adapters/slacksearch | not run |
| AC-026.2 | planned | test/unit/kernel, test/unit/adapters/tempo, test/unit/adapters/slacksearch | not run |
| AC-026.3 | planned | test/unit/kernel, test/unit/adapters/tempo, test/unit/adapters/slacksearch | not run |
| AC-027.1 | planned | make ci (coverage step) | not run |
| AC-027.2 | planned | make ci (coverage step) | not run |
| AC-027.3 | planned | make ci (coverage step) | not run |
| AC-028.1 | planned | test/unit/timesheet/domain | not run |
| AC-028.2 | `TestMarkEntrySubmittedKeepsFirstWorklogID` | test/unit/timesheet/domain | skipped (BUG exactly-once; un-skip with US-028) |
| AC-029.1 | planned | test/unit/timesheet/domain, test/unit/timesheet/application, test/unit/adapters/tempo, test/unit/kernel, test/architecture | not run |
| AC-029.2 | planned | test/unit/timesheet/domain, test/unit/timesheet/application, test/unit/adapters/tempo, test/unit/kernel, test/architecture | not run |
| AC-029.3 | planned | test/unit/timesheet/domain, test/unit/timesheet/application, test/unit/adapters/tempo, test/unit/kernel, test/architecture | not run |
| AC-029.4 | planned | test/unit/timesheet/domain, test/unit/timesheet/application, test/unit/adapters/tempo, test/unit/kernel, test/architecture | not run |
| AC-029.5 | planned | test/unit/timesheet/domain, test/unit/timesheet/application, test/unit/adapters/tempo, test/unit/kernel, test/architecture | not run |
