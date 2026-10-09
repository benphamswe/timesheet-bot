---
paths:
  - "services/timesheet-core/Dockerfile"
  - "docker-compose.yml"
  - "services/timesheet-core/.dockerignore"
---

# Container files

These rules cover the Go service `services/timesheet-core`. A container rule for the Python service `services/allocation-agent` comes with P3.

## Never

- Put a secret value in `ENV`, `ARG`, a layer, or `docker-compose.yml`. Compose passes variables through by name only.
- Run as root, or add a shell or package manager to the runtime image.

## Always

- Keep a multi-stage build ending in `gcr.io/distroless/static-debian12:nonroot`, with an exec-form `ENTRYPOINT`.
- Pin base images to a digest when you change them (`image:tag@sha256:...`).
- Keep the Go image's major and minor version equal to the `go` line in `services/timesheet-core/go.mod`.
- Run `docker build -t timesheet-bot:dev services/timesheet-core` after a change.
