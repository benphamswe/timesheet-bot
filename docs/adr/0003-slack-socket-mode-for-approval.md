# ADR-0003: Receive Slack interactions over Socket Mode

- Status: Accepted (implementation: P2)
- Date: 2026-10-09

## Context

Approval buttons need Slack to call back into the bot. The bot runs on a laptop behind NAT without a public URL (NFR-10).

## Decision

Use Slack Socket Mode: the bot opens an outbound WebSocket using an app-level token (`xapp-`, `connections:write`). The DM also shows the CLI fallback (`timesheet approve --week ...`) because clicks made while the bot is offline are lost.

## Alternatives considered

- HTTP request URL with a tunnel (ngrok, Cloudflare Tunnel): extra moving part and a public endpoint.
- Polling DM replies: worse UX, no buttons.

## Consequences

- Three Slack tokens: user (`search:read`), bot (`chat:write`), app-level (Socket Mode).
- The process must stay running to receive clicks.
