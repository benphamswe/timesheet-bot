# Git and GitHub

Read-only git commands, how the owner reviews a change before it is committed, and the `gh` commands behind the GitHub Issues mirror of `docs/backlog/`.

**Every write needs the owner's approval of the exact content first.** That covers `git commit`, `git push` (including a new branch), and every `gh` command that creates, edits or closes anything. `.claude/settings.json` makes these `ask` permissions, and the `guard-bash` hook blocks AI attribution, `--no-verify`, `--author` and force pushes. Agents never run them on their own initiative.

Run every command on this page from the repository root. The remote is named `benphamse`, not `origin`.

## Read-only git

### Status

```bash
git status --short
```

- **What it does**: lists changed (`M`), added (`A`) and untracked (`??`) files.
- **When to use it**: before and after any change, and to scope a review (`/review-fanout` uses `git status --porcelain`).
- **Expected**: one line per changed path. Confirmed 2026-10-09.

### Diff

```bash
git diff                 # unstaged changes
git diff --stat          # one line per file, with counts
git diff --staged        # what the next commit would contain
git diff -- docs/        # limit to a path
```

- **What it does**: shows changes against the index (or, with `--staged`, the index against `HEAD`).
- **When to use it**: to show the owner the exact change before asking to commit.
- **Expected**: a unified diff. `git diff --stat` confirmed 2026-10-09.

### Log

```bash
git log --oneline -10
git log --oneline -- services/timesheet-core/migrations/
```

- **What it does**: lists recent commits, optionally for one path.
- **When to use it**: to match the repository's commit message style before drafting one.
- **Expected**: `5daec83 first commit` on 2026-10-09.

### Remote

```bash
git remote -v
```

- **Expected**: `benphamse git@benphamse:benphamswe/timesheet-bot.git` for fetch and push. Confirmed 2026-10-09.

### What git ignores

```bash
git check-ignore -v services/timesheet-core/.env services/allocation-agent/.env \
  services/allocation-agent/.pytest_cache services/allocation-agent/.ruff_cache
git check-ignore -v services/allocation-agent/.env.example
git ls-files | grep -i env
```

- **Run from**: the repository root.
- **What it does**: prints the `.gitignore` rule that matches each path, without reading any file. The second form shows that `.env.example` templates stay committable (the match is the negation `!.env.example`). `git ls-files` lists the env templates git tracks, by name only.
- **When to use it**: after changing `.gitignore`, or before creating a new env file.
- **Expected** (2026-10-10): `.gitignore:3:.env` for both service `.env` paths, `.gitignore:19:.pytest_cache/` and `.gitignore:20:.ruff_cache/` for the caches, `.gitignore:5:!.env.example` for the template; `git ls-files` lists `services/timesheet-core/.env.example` and `timesheet-bot-env.example`.

### Scope a commit without reading env files

```bash
git status --short --ignored
git diff --stat -- . ':(exclude)**/.env*'
rtk proxy git diff -- <path>
git log -3 --format='%B---'
```

- **Run from**: the repository root.
- **What it does**: `--ignored` also lists ignored paths (`!!`) by name, so you can see build output, caches and `.venv` that git skips. The `:(exclude)**/.env*` pathspec keeps every `.env*` file out of the diff. `rtk proxy git diff` prints the raw unified diff when the `rtk` filter shortens it to a summary. `git log --format='%B---'` prints the full message of recent commits, to copy their convention.
- **When to use it**: when preparing a commit proposal, to flag files that should not be committed (by name only) and to match the message style.
- **Expected** (2026-10-10): `!!` for `services/allocation-agent/.venv/`, the `__pycache__`, `.pytest_cache` and `.ruff_cache` folders, `services/timesheet-core/bin/`, `coverage.out` and `config/config.yaml`; no `!!` line for any weekly file under `data/activity/`.

## How the owner reviews before committing

1. The agent finishes the change and runs the checks it touched (`make ci` from the root; see [CI](ci.md)).
2. The agent shows `git status --short` and `git diff` (or `git diff --stat` plus the diff of each file), and the proposed commit message in the repository's style. No `Co-Authored-By` or other AI attribution.
3. The owner reads the diff and the message and approves them, or asks for changes.
4. Only then is the commit made, with the repository's configured identity:

   ```bash
   git add <paths>
   git commit -m "<approved message>"
   ```

5. Pushing is a separate approval:

   ```bash
   git push benphamse <branch>
   ```

Steps 4 and 5 were **not run while documenting**. Approving one commit does not approve the next one, or the push.

## GitHub CLI for the backlog mirror

`docs/backlog/` is the source of truth; GitHub Issues is a mirror maintained by `/backlog-sync` through the `backlog-manager` agent (see [backlog](../backlog/README.md) and [Claude Code tooling](claude-code-tooling.md#slash-commands)). Use the slash command instead of running writes by hand.

All `gh` commands use the owner's GitHub login, so none were run while documenting.

### Reads (allowed without asking)

```bash
gh auth status
gh repo view
gh issue list --label type:story
gh issue list --milestone "SP-01: <goal>"
gh issue view <number>
gh label list
```

- **What they do**: check the login and repository, and read issues and labels.
- **When to use them**: `backlog-manager` runs them in mode `draft` to compare the backlog with GitHub.
- **Common failure**: if `gh auth status` reports no login, or `gh repo view` cannot resolve the repository, `/backlog-sync` stops and reports it. Run `gh auth login` yourself.
- **Not run while documenting.**

### Writes (owner approval of the draft required)

```bash
gh label create type:story --description "User story"
gh issue create --title "[US-001] ..." --label type:story --milestone "SP-01: <goal>" --body-file <file>
gh issue create --title "[FE-001] ..." --label type:feature --parent <epic-number> --body-file <file>
gh issue edit <number> --body-file <file>
gh issue close <number>
```

- **What they do**: create labels and issues (`--parent` makes a sub-issue), and update or close issues.
- **When to use them**: only in `/backlog-sync` mode `apply <sha256>`, which runs exactly the command lines in the approved `docs/backlog/.sync/draft.md` and refuses if the draft's sha256 changed after approval.
- **Not run while documenting.** The exact flags are written into each draft after `official-source-verifier` checks them against the gh manual.
