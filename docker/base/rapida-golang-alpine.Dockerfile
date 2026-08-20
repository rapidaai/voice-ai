# syntax=docker/dockerfile:1
# rapidaai/rapida-golang:1.25.13-alpine
# Extends golang:1.25.13-alpine — pinned base for all Go service builder stages.
# Published to: docker.io/rapidaai/rapida-golang:1.25.13-alpine
# Rebuild + push only when Go version changes: make push-rapida-golang-alpine
FROM golang:1.25.13-alpine
