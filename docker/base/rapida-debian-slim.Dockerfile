# syntax=docker/dockerfile:1
# rapidaai/rapida-debian:bookworm-slim
# Extends debian:bookworm-slim with common runtime deps and rapida-app user pre-configured.
# Published to: docker.io/rapidaai/rapida-debian:bookworm-slim
# Rebuild + push only when Debian version changes: just push-rapida-debian-slim
FROM debian:bookworm-slim

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates wget netcat-openbsd

RUN groupadd -g 1000 rapida-app && useradd -m -u 1000 -g rapida-app rapida-app

WORKDIR /opt/apps

RUN mkdir -p /opt/apps/assets /var/log/go-app && \
    chown -R rapida-app:rapida-app /opt/apps /var/log/go-app
