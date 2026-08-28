# syntax=docker/dockerfile:1
# rapidaai/rapida-alpine:3.21
# Extends alpine:3.21 with common runtime deps and rapida-app user pre-configured.
# Published to: docker.io/rapidaai/rapida-alpine:3.21
# Rebuild + push only when Alpine version changes: just push-rapida-alpine
FROM alpine:3.21

RUN apk --no-cache add ca-certificates wget netcat-openbsd && \
    addgroup -g 1000 rapida-app && \
    adduser -D -u 1000 -G rapida-app rapida-app

WORKDIR /opt/apps

RUN mkdir -p /opt/apps/assets /var/log/go-app && \
    chown -R rapida-app:rapida-app /opt/apps /var/log/go-app
