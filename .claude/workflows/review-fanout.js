export const meta = {
  name: 'review-fanout',
  description: 'Review uncommitted timesheet-bot changes per bounded context with ddd-reviewer and standards-reviewer, then adversarially verify every finding',
  whenToUse: 'After coding, before the owner commits. Args: optional {contexts: ["timesheet", ...]} to limit scope.',
  phases: [
    { title: 'Scope', detail: 'group changed files by bounded context' },
    { title: 'Review', detail: 'ddd-reviewer + standards-reviewer per context' },
    { title: 'Verify', detail: 'two skeptics try to refute each finding' },
  ],
}

const SCOPE_SCHEMA = {
  type: 'object',
  properties: {
    groups: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          context: { type: 'string' },
          files: { type: 'array', items: { type: 'string' } },
        },
        required: ['context', 'files'],
      },
    },
  },
  required: ['groups'],
}

const FINDINGS_SCHEMA = {
  type: 'object',
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          rule: { type: 'string' },
          file: { type: 'string' },
          line: { type: 'integer' },
          problem: { type: 'string' },
          scenario: { type: 'string' },
          fix: { type: 'string' },
          source: { type: 'string' },
        },
        required: ['rule', 'file', 'line', 'problem', 'scenario', 'fix'],
      },
    },
    uncovered: {
      type: 'array',
      items: {
        type: 'object',
        properties: { pattern: { type: 'string' }, source: { type: 'string' } },
        required: ['pattern', 'source'],
      },
    },
  },
  required: ['findings'],
}

const VERDICT_SCHEMA = {
  type: 'object',
  properties: { refuted: { type: 'boolean' }, reason: { type: 'string' } },
  required: ['refuted', 'reason'],
}

phase('Scope')
const wanted = (args && args.contexts) || null
const scope = await agent(
  'In the timesheet-bot repository, list uncommitted changed files (`git status --porcelain`, include untracked). ' +
  'Go code lives in services/timesheet-core/; a bounded context is the directory under services/timesheet-core/internal/. ' +
  'Group them by bounded context: timesheet, allocation, activity, learning, shared (services/timesheet-core/internal/shared), ' +
  'interfaces (services/timesheet-core/internal/interfaces and services/timesheet-core/cmd), ' +
  'platform (Go packages services/timesheet-core/internal/platform and internal/config), ' +
  'tests (every Go test under services/timesheet-core/test/, including test/architecture; ADR-0009), ' +
  'repo (services/*/Dockerfile, docker-compose.yml, services/timesheet-core/migrations, Makefile and services/*/Makefile, ' +
  'services/timesheet-core/.golangci.yml, .github/, api/proto/, services/allocation-agent/), docs (docs/, README.md). ' +
  'Ignore .claude/ files. Return only groups with at least one file.',
  { label: 'scope', phase: 'Scope', schema: SCOPE_SCHEMA, effort: 'low' },
)
const groups = scope.groups.filter(g => !wanted || wanted.includes(g.context))
log(`${groups.length} context(s) to review: ${groups.map(g => g.context).join(', ') || 'none'}`)

const REVIEWERS = [
  { type: 'ddd-reviewer', applies: g => !['docs', 'repo'].includes(g.context) },
  { type: 'standards-reviewer', applies: () => true },
]

const jobs = groups.flatMap(g => REVIEWERS.filter(r => r.applies(g)).map(r => ({ g, r })))

const results = await pipeline(
  jobs,
  job => agent(
    `Review these changed files in the "${job.g.context}" context:\n${job.g.files.join('\n')}\n` +
    'Report verified findings only, and fill `uncovered` with patterns no rule covers (with an official source URL or "no official source").',
    { label: `${job.r.type}:${job.g.context}`, phase: 'Review', agentType: job.r.type, schema: FINDINGS_SCHEMA },
  ),
  (review, job) => {
    if (!review) return null
    return parallel(review.findings.map(f => () =>
      parallel([0, 1].map(i => () => agent(
        `Skeptic ${i + 1}: try to REFUTE this code review finding in the timesheet-bot repository. ` +
        'Read the cited code and its callers and tests. Refuted means the scenario cannot happen, the rule does not apply, ' +
        'or it contradicts docs/architecture/code-map.md, docs/development/conventions.md or an accepted ADR. Default to refuted=true if you cannot reproduce the reasoning from the code.\n' +
        JSON.stringify(f),
        { label: `verify:${f.file}:${f.line}#${i + 1}`, phase: 'Verify', schema: VERDICT_SCHEMA },
      ))).then(votes => {
        const ok = votes.filter(Boolean).filter(v => !v.refuted).length
        return { ...f, reviewer: job.r.type, context: job.g.context, survives: ok === 2, votes: votes.filter(Boolean) }
      }),
    )).then(verified => ({ verified, uncovered: review.uncovered || [] }))
  },
)

const done = results.filter(Boolean)
const all = done.flatMap(r => r.verified)
const confirmed = all.filter(f => f.survives)
const rejected = all.filter(f => !f.survives).map(f => ({ rule: f.rule, file: f.file, line: f.line, reasons: f.votes.map(v => v.reason) }))
const uncovered = done.flatMap(r => r.uncovered)
log(`${confirmed.length} confirmed, ${rejected.length} refuted, ${uncovered.length} uncovered pattern(s)`)
if (jobs.length > done.length) log(`${jobs.length - done.length} review job(s) failed and are not covered`)

return { confirmed, rejected, uncovered }
