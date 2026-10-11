# syntax=docker/dockerfile:1
# Slack activity collector (`collect`, ADR-0011). Used by the `collector` service in
# docker-compose.yml (profile "collect"). No secret enters the image: .dockerignore excludes .env.
#
# The claude-agent-sdk wheel bundles the Claude Code CLI as a glibc binary, so the base image is
# Debian (slim), not Alpine, and no Node.js is needed.

# uv, pinned to [tool.uv] required-version in pyproject.toml.
FROM ghcr.io/astral-sh/uv:0.12.21@sha256:a7aed3216253ee804de3e2d8afa5073baa1a177335345d43845cd4165e43b711 AS uv

FROM python:3.12-slim-bookworm@sha256:34386ef0cb081344d7ec1c103ba398e6e9f64e9ab3a1509accc92a4e24a07258 AS build
COPY --from=uv /uv /bin/uv
# The image mirrors the repository layout under /repo (see the runtime stage).
ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_PYTHON_DOWNLOADS=never
WORKDIR /repo/services/allocation-agent
# Dependencies first, so a source change does not reinstall them. Installs never re-resolve.
COPY pyproject.toml uv.lock .python-version ./
RUN uv sync --locked --no-dev --no-install-project
COPY src ./src
RUN uv sync --locked --no-dev --no-editable

FROM python:3.12-slim-bookworm@sha256:34386ef0cb081344d7ec1c103ba398e6e9f64e9ab3a1509accc92a4e24a07258
# uid/gid 1000 is the default WSL user, so week files written to the bind-mounted
# data/activity stay owned by the owner on the host.
ARG APP_UID=1000
ARG APP_GID=1000
# The collector finds the repository root by walking up from its own install path to a folder
# with services/timesheet-core/ and a Makefile (config.find_repo_root). The empty Makefile is
# only that marker. Compose mounts config/ and data/activity/ into these folders.
RUN groupadd --gid "${APP_GID}" app \
 && useradd --uid "${APP_UID}" --gid app --home-dir /home/app --create-home --shell /usr/sbin/nologin app \
 && mkdir -p /repo/services/timesheet-core/config /repo/data/activity \
 && touch /repo/Makefile \
 && chown app:app /repo/data/activity
COPY --from=build /repo/services/allocation-agent/.venv /repo/services/allocation-agent/.venv
ENV PATH="/repo/services/allocation-agent/.venv/bin:${PATH}" \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1
WORKDIR /repo/services/allocation-agent
USER app:app
ENTRYPOINT ["collect"]
CMD ["--help"]
