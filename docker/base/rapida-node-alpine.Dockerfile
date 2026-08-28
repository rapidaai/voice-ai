# syntax=docker/dockerfile:1
# rapidaai/rapida-node:22-alpine
# Extends node:22-alpine with common build tools.
# Published to: docker.io/rapidaai/rapida-node:22-alpine
# Rebuild + push only when Node version changes: just push-rapida-node-alpine
FROM node:22-alpine

RUN apk add --no-cache curl
