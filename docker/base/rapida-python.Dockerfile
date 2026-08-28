# syntax=docker/dockerfile:1
# rapidaai/rapida-python:3.11
# Extends tiangolo/uvicorn-gunicorn:python3.11 with system build deps and rapida-app user.
# Published to: docker.io/rapidaai/rapida-python:3.11
# Rebuild + push only when Python version or system deps change: just push-rapida-python
FROM tiangolo/uvicorn-gunicorn:python3.11

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    automake \
    autoconf \
    libtool \
    pkg-config \
    cmake \
    git \
    wget \
    curl \
    libheif-dev \
    libde265-dev \
    libjpeg-dev \
    zlib1g-dev \
    libtiff-dev \
    libopenjp2-7-dev

RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --upgrade pip setuptools wheel

RUN addgroup --system rapida-app && adduser --system --group rapida-app
