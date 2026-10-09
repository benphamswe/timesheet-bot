# standards-curator memory

## Fetch failures

- netflixtechblog.com: HTTP 403 to the fetcher (2026-10-09). Use github.com/Netflix wikis and READMEs instead.
- aws.amazon.com/builders-library/* now 301-redirects to builder.aws.com/content/...; cite the new URL.
- azure.github.io/azure-sdk/golang_design.html: 404; design rules live on golang_introduction.html.
- go.dev/blog/errors: 404; use go.dev/blog/go1.13-errors.
- google.aip.dev/client-libraries/4221: fetch returns only a redirect stub (2026-10-09).
- golangci-lint.run/docs/linters/configuration/: too long to reach the sloglint block; read github.com/go-simpler/sloglint README instead.
- Pages without anchor ids (cite URL only): go.dev/doc/security/best-practices, go.dev/doc/modules/layout, genai.owasp.org LLM01, builder.aws.com posts, eng-practices reviewer/standard.html. Microsoft Learn anchors were not confirmed either.

## Organizations with no official public standard for this stack

Checked 2026-10-09; do not search again unless the scope changes.

- Meta: no company-wide coding standard or Go/backend guide (only per-project contributing guides).
- Apple: nothing for Go/backend; Swift.org API Design Guidelines are Swift-only.
- Cloudflare: only a docs content style guide.
- Shopify: only a Ruby style guide.
- Spotify: only blog posts and product docs.
- LinkedIn: none; Rest.li is a framework, repo archived 2026-07.
- HashiCorp: no Go style guide; Terraform provider best practices are for plugin authors only.
- Twitter/X: only Effective Scala (Scala, stale 2023).
- Google: no SQL/BigQuery style guide.
- Microsoft SDL: practice-level only, no citable code rules.

## Rules the owner rejected

(none yet)

## Edition changes

- OWASP LLM Top 10: cite the 2026 edition (github.com/GenAI-Security-Project/GenAI-LLM-Top10/tree/main/2026/final). genai.owasp.org/llmrisk pages still show the 2025 edition. Excessive Agency moved from LLM06 to LLM03.
- OWASP ASVS 5.0 renumbered chapters: architecture V15, logging and errors V16, secrets V13.3.
