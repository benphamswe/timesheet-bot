# ADR-0013: Keep one .env file per service instead of one at the repository root

- Status: Proposed
- Date: 2026-10-10
- Deciders: owner

## Context

- Until now, one repository-root `.env` held the local settings and, optionally, the tokens of every service. `make -C services/timesheet-core run` sourced `../../.env`, and compose used the same file for `${VAR}` interpolation and as the `app` service's `env_file`.
- The monorepo has two services (ADR-0008) with different secrets:
  - `services/timesheet-core` (Go CLI): `SLACK_USER_TOKEN`, `JIRA_*`, `TEMPO_API_TOKEN`, `MAVENLINK_*`, `DATABASE_URL`.
  - `services/allocation-agent` (Python collector, ADR-0011): no Claude credential at all, because it must run on the owner's claude.ai login. Later, `DEEPSEEK_API_KEY` when `report.parser` is `llm`.
- A shared file gives each process every other service's tokens. Worse, the collector must never see `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN`: with any of them active, Claude Code does not load claude.ai connectors, so the Slack connector disappears. For that reason the collector loaded no `.env` at all.
- The `app` service passed secrets through `environment:` as `KEY: ${KEY}` and `KEY: ${KEY:-}`. Compose interpolates `${KEY}` from the shell and the project-root `.env` only. With the tokens moved into a service file, that interpolation would yield an empty string, and `environment:` beats `env_file` "even if those values are empty or undefined", so the token loaded by `env_file` would be blanked.

## Decision

- Each service owns one `.env` in its own folder, written by the owner only, gitignored (the `.env` rule in `.gitignore` matches at any depth), mode `0600`:
  - `services/timesheet-core/.env`, read by the Go CLI;
  - `services/allocation-agent/.env`, read by the Python collector.
  The repository-root `.env` is no longer read by any `make` target and no longer referenced by `docker-compose.yml`.
- Host runs load the service's own file with the same shell-wins macro (`LOAD_DOTENV` in both service Makefiles: export the file, then restore every variable already exported in the shell). A variable exported in `~/.zshrc` wins; an empty `KEY=` never clears it. `DOTENV := ./.env`; the `./` stops a POSIX shell from searching `PATH` for the file.
- `make collect` now loads `services/allocation-agent/.env`. `collect` checks, in Python at startup and before any Claude session, that `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` are empty, and exits 2 naming the variables that are set (names only, never values). `DEEPSEEK_API_KEY` is not checked.
- Compose:
  - `app` loads `services/timesheet-core/.env` with `env_file` (long syntax, `required: false`, so `docker compose up -d db` works without it).
  - `environment:` holds no secret in any form, neither `${KEY}`, `${KEY:-}` nor a bare `KEY:`. It keeps only non-secret, container-specific values: `DATABASE_URL` pointing at host `db`.
  - A token exported only in the shell reaches the container for one run with `docker compose run --rm -e KEY app ...`; an unset `KEY` leaves the `env_file` value in place.
  - `db` uses no interpolation and needs no env file.
  - A new `collector` service (profile `collect`) builds `services/allocation-agent` with uv, loads `services/allocation-agent/.env` (optional) as its only secret source, mounts `data/activity` read-write and `services/timesheet-core/config` read-only, runs non-root on a read-only root filesystem with no capabilities, on its own network with no route to `db`.
- The collector container gets no Claude credential until the owner decides how (see [Collector credentials in a container](#collector-credentials-in-a-container)). Until then, the documented way to collect is host-run `make collect`.

## Compose interpolation, checked

Checked on 2026-10-10 with dummy values in a scratch compose file (Compose v5.0.2): a service file holding `FOO`, `BAR` and `BAZ`, with `environment:` set to `FOO:` and `BAR: ${BAR:-}`, and the shell setting none of them.

| `environment:` entry | Value in the container |
|---|---|
| `BAR: ${BAR:-}` | `BAR=` (blanked) |
| `FOO:` (no value) | unset (the `env_file` value is hidden) |
| none (`BAZ`) | the `env_file` value |
| none, `run -e BAZ` with `BAZ` exported in the shell | the shell value |
| none, `run -e BAZ` with `BAZ` unset in the shell | the `env_file` value |

Official sources: `env_file` and its `required` attribute, and "Environment variables declared in the `environment` section override these values. This holds true even if those values are empty or undefined" (<https://docs.docker.com/reference/compose-file/services/#env_file>). Interpolation reads the shell, then `--env-file`, then the project-directory `.env` (<https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/>).

## Collector credentials in a container

The claude.ai Slack connector is fetched only when Claude Code's active credential is a claude.ai subscription login. Facts from the official docs, read on 2026-10-10:

- On Linux, Claude Code stores credentials in `~/.claude/.credentials.json` with mode `0600`. With `CLAUDE_CONFIG_DIR` set, it keeps `.credentials.json` under that directory instead (<https://code.claude.com/docs/en/authentication#credential-management>).
- The OAuth account also lives in `~/.claude.json`, outside `~/.claude`. To persist a login in a container, mount a volume at the config directory and set `CLAUDE_CONFIG_DIR` to the same path, "so Claude Code writes `.claude.json` inside the volume" (<https://code.claude.com/docs/en/devcontainer#persist-authentication-and-settings-across-rebuilds>).
- The saved login is renewed, that is rewritten, by Claude Code: "Parallel sessions on one machine share a saved login and coordinate its renewal so that only one process refreshes the token at a time", and two sessions renewing with the same token once "revoked the saved login" (<https://code.claude.com/docs/en/troubleshoot-install#not-logged-in-or-token-expired>). A read-only credential mount therefore stops working once the access token expires.
- Connectors are not loaded when `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` or `apiKeyHelper` is active, when an Anthropic profile supplies the credential, or when `CLAUDE_CODE_OAUTH_TOKEN` holds a `claude setup-token` token, "which can only make model requests" (<https://code.claude.com/docs/en/mcp#use-mcp-servers-from-claude-ai>).
- `CLAUDE_CODE_OAUTH_REFRESH_TOKEN` with `CLAUDE_CODE_OAUTH_SCOPES` lets `claude auth login` exchange a refresh token "directly instead of opening a browser" (<https://code.claude.com/docs/en/env-vars>). Whether the resulting login loads claude.ai connectors is not documented.
- The dev container page warns that a compromised session can exfiltrate "the Claude Code credentials stored in `~/.claude`", and advises against mounting host secrets (<https://code.claude.com/docs/en/devcontainer>).

These were read directly from code.claude.com by the agent that wrote this ADR; `official-source-verifier` has not run on them yet.

Options for the owner:

| Option | How | Security trade-off | Works? |
|---|---|---|---|
| A. Host-run `make collect` (current) | No container for collection | Nothing new exposed; the login stays in the owner's `~/.claude` | Yes |
| B. Dedicated named volume, separate login | Named volume (for example `claude-collector`) at `/home/app/.claude`, `CLAUDE_CONFIG_DIR=/home/app/.claude`; one interactive `/login` inside the container with the bundled CLI | A second full-scope subscription login, isolated from the host's. It sits in plaintext in a Docker volume, readable by anyone with Docker access on the host; revocable on its own. Only the collector container mounts it. The renewal stays inside one config directory, so it cannot revoke the host login. | Expected; not yet tested |
| C. Bind-mount the owner's `~/.claude` and `~/.claude.json` read-write | Bind mounts from the host | Exposes every file under `~/.claude` (all session history, settings, other MCP credentials) to the container. Host and container processes renew the same token, possibly without coordination across the container boundary, risking revocation of the owner's main login. | Not recommended |
| D. Bind-mount the credentials read-only | `:ro` mounts | Same exposure as C, and renewal cannot write, so collection fails once the access token expires | No |
| E. `CLAUDE_CODE_OAUTH_TOKEN` from `claude setup-token` | `.env` entry | One-year token, but it "can only make model requests" and cannot fetch connectors; `collect` also refuses it | No |
| F. `ANTHROPIC_API_KEY` | `.env` entry | Hides the connectors; `collect` refuses it | No |
| G. Refresh-token provisioning | `CLAUDE_CODE_OAUTH_REFRESH_TOKEN` + `CLAUDE_CODE_OAUTH_SCOPES`, then `claude auth login` into a volume as in B | Puts a long-lived refresh token into an env file; connector support undocumented | Unverified |

Recommendation: keep A. If containerised collection is needed, B is the least-exposure option; verify it with the connector check in [activity collection](../commands/activity-collection.md) before relying on it. `docker-compose.yml` holds a commented placeholder and no credential mount.

## Egress

- The collector's Claude session reaches Anthropic (the Claude API and claude.ai for the connector).
- When `report.parser` is `llm`, the collector also calls `api.deepseek.com` over HTTPS 443 with `DEEPSEEK_API_KEY`, sending the report summaries to a third party. The owner confirms company policy before enabling it.
- The `collect` compose network has no egress restriction; it only separates the collector from `db`. See [network](../architecture/network.md).

## Alternatives considered

- Keep one root `.env`: every process sees every token, and the collector cannot load it without risking `ANTHROPIC_API_KEY`.
- Keep `KEY: ${KEY}` passthrough in `environment:` and point compose at the service file with `--env-file`: `--env-file` replaces the project `.env` for interpolation of the whole compose file, so one file per service is not possible, and every command would need the flag.
- Load `.env` inside the programs (a dotenv library): adds a dependency in each language and a second precedence rule; the Makefile macro already does it with the shell winning.

## Consequences

- Positive:
  - Each service sees only its own secrets; the collector never inherits Go-core tokens.
  - No secret can be blanked by an interpolated `environment:` entry.
  - A misplaced Anthropic credential stops `collect` at once with a clear message instead of a missing-connector failure.
- Negative / accepted risk:
  - Tokens exported only in `~/.zshrc` no longer reach the `app` container automatically; pass them with `docker compose run -e KEY`, or put them in the service `.env`.
  - `services/timesheet-core/.env` now sits inside the Go image's build context. `services/timesheet-core/.dockerignore` excluding `.env` and `.env.*` is now the control that keeps it out of the image, not just defence in depth. `services/allocation-agent/.dockerignore` does the same.
  - Two files to keep at `chmod 600` instead of one.
  - Compose still reads a project-root `.env` for interpolation if one exists. Nothing in `docker-compose.yml` interpolates any more, but the owner should delete the root `.env` after moving its values.
- Follow-up (owner):
  - Move the values from the root `.env` into the two service files, `chmod 600` both, then delete the root `.env`.
  - Create the `.env.example` templates (agents cannot write them): `services/allocation-agent/.env.example`, update `services/timesheet-core/.env.example`, and replace or delete the root `timesheet-bot-env.example`.
  - Update `CLAUDE.md` (Non-negotiable: "the owner's gitignored root `.env`") and `.claude/rules/container.md` ("Compose passes variables through by name only"); agents do not edit those.
  - Decide the collector credential option above.
  - Phase 3: the allocation agent's `ANTHROPIC_API_KEY` (ADR-0007) must not go into `services/allocation-agent/.env` while `collect` reads that file; give the P3 service its own env file or secret.
