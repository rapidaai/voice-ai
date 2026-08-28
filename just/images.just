go_version := "1.25.13"

# Build and push the Go Bookworm base image.
push-rapida-golang-bookworm:
    env DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-golang-bookworm.Dockerfile -t rapidaai/rapida-golang:{{go_version}}-bookworm .
    docker push rapidaai/rapida-golang:{{go_version}}-bookworm

# Build and push the Go Alpine base image.
push-rapida-golang-alpine:
    env DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-golang-alpine.Dockerfile -t rapidaai/rapida-golang:{{go_version}}-alpine .
    docker push rapidaai/rapida-golang:{{go_version}}-alpine

# Build and push the Alpine base image.
push-rapida-alpine:
    env DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-alpine.Dockerfile -t rapidaai/rapida-alpine:3.21 .
    docker push rapidaai/rapida-alpine:3.21

# Build and push the Debian base image.
push-rapida-debian-slim:
    env DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-debian-slim.Dockerfile -t rapidaai/rapida-debian:bookworm-slim .
    docker push rapidaai/rapida-debian:bookworm-slim

# Build and push the Node base image.
push-rapida-node-alpine:
    env DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-node-alpine.Dockerfile -t rapidaai/rapida-node:22-alpine .
    docker push rapidaai/rapida-node:22-alpine

# Build and push the Python base image.
push-rapida-python:
    env DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-python.Dockerfile -t rapidaai/rapida-python:3.11 .
    docker push rapidaai/rapida-python:3.11

# Build and push every base image.
push-base-images: push-rapida-golang-bookworm push-rapida-golang-alpine push-rapida-alpine push-rapida-debian-slim push-rapida-node-alpine push-rapida-python
