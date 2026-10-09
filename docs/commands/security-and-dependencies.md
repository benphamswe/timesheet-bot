# Security and dependencies

Vulnerability scanning, module hygiene, and the read-only lookups used to check a technology choice (the evidence for `.claude/skills/tech-selection/SKILL.md`). Every lookup on this page only reads public data; none needs a secret.

Run the Go commands from `services/timesheet-core/`. The `curl` and `git ls-remote` lookups work from any folder.

## Vulnerabilities

### `govulncheck`

```bash
make vuln                  # go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
govulncheck ./...          # a locally installed govulncheck, any version
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -version   # scanner, Go and vulnerability database versions
```

- **What it does**: reports known vulnerabilities in the Go standard library and modules that your code actually reaches, using `https://vuln.go.dev`. `make vuln` runs the version pinned in `GOVULNCHECK_VERSION` (service `Makefile`) with `go run`, so no local install is needed and the version cannot drift. CI runs `make vuln` too. A plain `govulncheck` runs whatever version you installed.
- **When to use it**: before every review, after any `go.mod` change, and after bumping the toolchain.
- **Expected**: `No vulnerabilities found.` Before it, `go` prints `go: golang.org/x/vuln@v1.8.0 requires go >= 1.26.0; switching to go1.26.9` on stderr; that line is normal and means the scan runs on the module toolchain. `-version` prints `Go: go1.26.9` and `Scanner: govulncheck@v1.8.0`. Confirmed 2026-10-09.
- **Common failure**: a finding names the module, the fixed version and the call path. Upgrade to the fixed version (`go get <module>@<fixed>`, then `go mod tidy`) on a branch and rerun.

### From the repository root

```bash
make vuln                                                                     # root form: runs inside each service; CI uses this
go -C services/timesheet-core run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...   # root form: the same command without make
```

- **Expected**: `== services/timesheet-core: vuln` (make form only), then `No vulnerabilities found.` Both confirmed 2026-10-09.
- **Warning: false alarm with `govulncheck -C`**. Do not use it; the project permissions no longer allow it. `govulncheck -C services/timesheet-core ./...` analyses the standard library of the Go it finds from the root, which is the system Go (`go1.22.10` on the owner's machine), not the pinned `go1.26.9`. On 2026-10-09 it reported `net/http@go1.22.10` vulnerabilities (for example `GO-2026-6611` and `GO-2026-6617`) and exited 3, although the project is clean. The `go -C services/timesheet-core run` form above uses the module toolchain instead: `go -C services/timesheet-core run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -version` prints `Go: go1.26.9` (confirmed 2026-10-09).

## Module hygiene

### `go mod tidy`

```bash
go mod tidy -diff   # show what tidy would change; edits nothing
go mod tidy         # rewrite go.mod and go.sum
```

- **What it does**: adds missing and removes unused requirements in `go.mod` and `go.sum`.
- **When to use it**: after adding or removing an import of a third-party package.
- **Expected**: `-diff` prints nothing and exits 0. Confirmed 2026-10-09. The rewrite form was not run while documenting, because it edits `go.mod`.

### Check for module updates

```bash
# direct dependencies with a newer version
go list -m -u -f '{{if and (not .Indirect) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all

# the whole module graph
go list -m -u all
```

- **What it does**: asks the module proxy for newer versions. The plain form lists every module in the graph, including the database drivers `goose` pulls in, so most of its lines are not yours to upgrade. The filtered form shows only direct requirements.
- **When to use it**: when planning an upgrade (`technical-architect`), or when Dependabot opens a PR and you want the full picture.
- **Expected**: on 2026-10-09 the filtered form printed nothing (every direct dependency is current); the plain form listed about 40 indirect modules with `[newer]` versions.

### Prove a package's imports

```bash
go list -deps ./internal/timesheet/domain | grep timesheet-bot
```

- **What it does**: lists every package the target depends on, transitively; the `grep` keeps the ones inside this module.
- **When to use it**: to prove or disprove a boundary violation (review checklist B1, `.claude/rules/domain.md`). Use it with `make arch`.
- **Expected**: only `timesheet-bot/internal/shared/kernel` and the package itself. Confirmed 2026-10-09.

## Version and supply-chain lookups

Use these when filling a plan's Technology decisions table. Claims about a vendor still go through `official-source-verifier`.

### Latest version of a Go module

```bash
curl -sS https://proxy.golang.org/github.com/spf13/cobra/@latest | jq .Version
```

- **What it does**: asks the Go module proxy for the latest release of a module. Write each capital letter in the module path as `!` plus the lowercase letter (for example `github.com/!burnt!sushi/toml`).
- **Expected**: `"v1.10.2"` on 2026-10-09.

### Latest Go toolchain

```bash
curl -sS 'https://go.dev/dl/?mode=json' | jq -r '.[0].version'                                         # newest stable release
curl -sS 'https://go.dev/dl/?mode=json' | jq -r '[.[] | select(.version|startswith("go1.26."))][0].version'   # newest 1.26 patch
```

- **What it does**: reads the official download list (stable releases only, newest first).
- **When to use it**: to check the `toolchain` line in `go.mod` and the `golang:` tag in the `Dockerfile` (criterion P12).
- **Expected** on 2026-10-09: `go1.27.2` and `go1.26.9`. The pinned toolchain is the newest 1.26 patch.
- **Common failure**: without quotes, zsh treats `?` as a glob and fails with `no matches found`. Keep the URL in single quotes.

### Latest version of a Python package

```bash
curl -sS https://pypi.org/pypi/langgraph/json | jq -r .info.version
```

- **What it does**: reads the PyPI JSON API for the package.
- **When to use it**: for `services/allocation-agent` choices [planned: P3].
- **Expected**: `1.2.14` on 2026-10-09.

```bash
pip index versions claude-agent-sdk     # every published version, newest first
```

- **What it does**: the same lookup through pip. It reads PyPI only and installs nothing. `uv pip` has no `index` subcommand.
- **Expected**: `claude-agent-sdk (0.2.165)` on 2026-10-10.

### Inspect a Python package's API in a scratch venv

```bash
mkdir -p /tmp/sdkprobe && cd /tmp/sdkprobe
uv venv -q --python 3.12 .venv && uv pip install -q --python .venv/bin/python claude-agent-sdk
.venv/bin/python -c "import claude_agent_sdk as s; print(s.__version__)"
cat .venv/lib/python3.12/site-packages/claude_agent_sdk/_cli_version.py
grep -n "setting_sources\|output_format\|strict_mcp_config" .venv/lib/python3.12/site-packages/claude_agent_sdk/types.py
```

- **Run from**: `/tmp`, outside the repository, so nothing in the project changes.
- **What it does**: installs the package into a throwaway venv to read its real option names, types and docstrings before you write code against it.
- **When to use it**: before you depend on an SDK field that the docs only describe in prose.
- **Expected** (2026-10-10): `0.2.165`, bundled CLI `2.1.294`. `ClaudeAgentOptions` has `setting_sources` (where `None` means all sources), `output_format`, and `strict_mcp_config`. `ResultMessage` has `structured_output`. `ClaudeSDKClient.get_mcp_status()` lists servers with `scope` and `tools`.

### License, advisories and publish date (deps.dev)

```bash
curl -sS "https://api.deps.dev/v3/systems/go/packages/github.com%2Fspf13%2Fcobra/versions/v1.10.2" \
  | jq -c '{licenses, advisoryKeys, publishedAt}'
```

- **What it does**: reads Google Open Source Insights for one version. URL-encode `/` in the module path as `%2F`. Use `systems/pypi/packages/<name>` for Python.
- **Expected**: `{"licenses":["Apache-2.0"],"advisoryKeys":[],"publishedAt":"2025-12-03T23:51:15Z"}` on 2026-10-09.

### Project health and OpenSSF Scorecard (deps.dev)

```bash
curl -sS "https://api.deps.dev/v3/projects/github.com%2Fspf13%2Fcobra" \
  | jq -c '{scorecard: .scorecard.overallScore, openIssuesCount}'
```

- **What it does**: reads the project record, including the OpenSSF Scorecard overall score.
- **Expected**: `{"scorecard":6.2,"openIssuesCount":464}` on 2026-10-09. A `null` score means deps.dev has no Scorecard for the project (as for `langgraph` in ADR-0007); record that as missing evidence.

### Advisories on one version (deps.dev)

```bash
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/langgraph-checkpoint/versions/4.1.0" | jq -c '{advisoryKeys}'
curl -sS "https://api.deps.dev/v3/advisories/GHSA-fjqc-hq36-qh5p" | jq -c '{title, aliases, cvss3Score}'
```

- **What it does**: the first form lists the advisory ids recorded against one version; the second reads one advisory's title, aliases (CVE, PYSEC) and CVSS score.
- **When to use it**: to check that the version a lockfile resolves is past every known advisory (criterion P1), including transitive packages such as `langgraph-checkpoint`.
- **Expected** on 2026-10-09: `{"advisoryKeys":[{"id":"GHSA-fjqc-hq36-qh5p"},{"id":"PYSEC-2026-2573"}]}`, then `{"title":"LangGraph Checkpoint: Unsafe JSON deserialization in checkpoint loading","aliases":["CVE-2026-48775","PYSEC-2026-2573"],"cvss3Score":6.8}`.
- **Common failure**: deps.dev can lag GitHub by days. On 2026-10-09 it showed no advisory on `langgraph` 1.2.13 although GHSA-3fx2-cqw9-xr3c was already published. Cross-check with the GitHub advisories lookup below.

### Resolved dependency graph (deps.dev)

```bash
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/langgraph/versions/1.2.14:dependencies" | jq '.nodes | length'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/langgraph/versions/1.2.14:dependencies" \
  | jq -c '[.nodes[] | select(.relation=="DIRECT") | .versionKey.name + " " + .versionKey.version]'
```

- **What it does**: reads the dependency graph deps.dev resolved for one version. The first form counts every node (the package itself included); the second lists only direct dependencies and the versions deps.dev resolved.
- **When to use it**: to judge footprint (criterion P8) and to see which transitive packages need their own advisory check.
- **Expected** on 2026-10-09: `38`, then `["langchain-core 1.6.7","langgraph-checkpoint 4.2.0","langgraph-prebuilt 1.1.0","langgraph-sdk 0.4.6","pydantic 2.13.5","xxhash 4.0.1"]`.

### Security advisories of a GitHub repository

```bash
curl -sS "https://api.github.com/repos/langchain-ai/langgraph/security-advisories?per_page=20" \
  | jq -c '.[] | {ghsa_id, severity, published_at, patched: [.vulnerabilities[].patched_versions]}'
```

- **What it does**: lists the repository's published GitHub Security Advisories with the patched versions. It works without credentials (unauthenticated calls are rate limited per IP).
- **When to use it**: to find the minimum safe version, and to catch advisories that deps.dev does not show yet.
- **Expected** on 2026-10-09: the first line is `{"ghsa_id":"GHSA-3fx2-cqw9-xr3c","severity":"medium","published_at":"2026-10-07T17:31:37Z","patched":["0.4.6","1.2.14"]}`, which makes `langgraph` 1.2.14 the floor. Each patched version belongs to one affected package (here `langgraph-sdk` 0.4.6 and `langgraph` 1.2.14); the advisory page names them.

### Declared dependencies of a Python package (PyPI `requires_dist`)

```bash
curl -sS https://pypi.org/pypi/langgraph/1.2.14/json | jq -c '.info.requires_dist'
curl -sS https://pypi.org/pypi/grpcio-tools/1.84.0/json | jq -c '.info.requires_dist'
```

- **What it does**: reads the version ranges a package declares for its dependencies (criterion P7). Unlike the deps.dev graph, this shows the allowed range, not one resolved version.
- **When to use it**: to check whether a declared range still admits a vulnerable version, which then needs its own floor in the lockfile.
- **Expected** on 2026-10-09: the `langgraph` list includes `"langgraph-checkpoint<5.0.0,>=4.1.0"`, which still admits the vulnerable 4.1.0, so the lockfile needs `langgraph-checkpoint>=4.1.1` ([ADR-0007](../adr/0007-grpc-contracts-now-split-later.md), condition C20). The `grpcio-tools` list is `["protobuf<8.0.0,>=7.35.1","grpcio>=1.84.0","setuptools>=77.0.1"]`.

### Resolve a GitHub Action tag to a commit SHA

Workflows pin actions by full commit SHA with the tag in a comment (`.github/workflows/ci.yml`).

```bash
git ls-remote --tags https://github.com/actions/checkout 'v7.0.1*'
```

- **What it does**: lists the tag from the public repository without credentials. For an annotated tag a second line ending in `^{}` appears; that line's SHA is the commit to pin.
- **Expected**: `3d3c42e5aac5ba805825da76410c181273ba90b1 refs/tags/v7.0.1` on 2026-10-09, which matches `ci.yml`. Same check confirmed for `actions/setup-go` `v7.0.0`.

```bash
gh api repos/actions/checkout/git/ref/tags/v7.0.1 --jq '.object.type + " " + .object.sha'
# if the type is "tag" (annotated), dereference it:
gh api repos/actions/checkout/git/tags/<tag-object-sha> --jq .object.sha
```

- **What it does**: the same lookup through the GitHub REST API with your `gh` login. It is a read (`GET`), but `gh api` is an `ask` permission in `.claude/settings.json`, so Claude Code prompts first.
- **Not run while documenting**: it uses the owner's GitHub credentials. Use `git ls-remote` above when you only need the SHA.
