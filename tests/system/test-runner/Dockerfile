# syntax=docker/dockerfile:1
FROM golang:1.25.13-alpine@sha256:1e0126852075c9c60731c8ba49088448b91f63e2aed97ca9d1a9791622a05946 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY tests/system ./tests/system
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -o /out/systemcheck ./tests/system/cmd/systemcheck

FROM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32
RUN apk add --no-cache ca-certificates postgresql-client
WORKDIR /workspace
COPY tests/system/test-runner/package.json tests/system/test-runner/package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY --from=builder /out/systemcheck /usr/local/bin/systemcheck
COPY openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json /workspace/openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json
RUN mkdir -p /reports && chown node:node /reports
USER node
CMD ["systemcheck"]
