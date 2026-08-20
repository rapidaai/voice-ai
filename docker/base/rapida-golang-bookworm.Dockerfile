# syntax=docker/dockerfile:1
# rapidaai/rapida-golang:1.25.13-bookworm
# Extends golang:1.25.13-bookworm with C libraries required by assistant-api.
# Published to: docker.io/rapidaai/rapida-golang:1.25.13-bookworm
# Rebuild + push only when SDK versions change: make push-rapida-golang-bookworm
FROM golang:1.25.13-bookworm AS base

WORKDIR /app

# Install system dependencies for CGO
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc g++ make autoconf automake libtool pkg-config \
    curl git wget unzip ca-certificates tar \
    libopus-dev libopusfile-dev

# ---- Install ONNX Runtime ----
RUN mkdir -p /tmp/build && cd /tmp/build && \
    wget -q https://github.com/microsoft/onnxruntime/releases/download/v1.16.0/onnxruntime-linux-x64-1.16.0.tgz && \
    mkdir -p /opt/onnxruntime && \
    tar -xzf onnxruntime-linux-x64-1.16.0.tgz -C /opt/onnxruntime --strip-components=1 && \
    rm onnxruntime-linux-x64-1.16.0.tgz

# ---- Build RNNoise ----
RUN cd /tmp/build && \
    curl -fsSL https://github.com/rapidaai/rnnoise/archive/refs/heads/main.tar.gz -o rnnoise.tar.gz && \
    tar -xzf rnnoise.tar.gz && \
    cd rnnoise-main && \
    ./autogen.sh && \
    ./configure --prefix=/usr/local && \
    make -j$(nproc) && \
    make install && \
    cd /tmp/build && \
    rm -rf rnnoise.tar.gz rnnoise-main

# ---- Install Azure Speech SDK ----
RUN cd /tmp/build && \
    wget -q -O SpeechSDK-Linux.tar.gz https://aka.ms/csspeech/linuxbinary && \
    mkdir -p /opt/azure-speech-sdk && \
    tar --strip 1 -xzf SpeechSDK-Linux.tar.gz -C /opt/azure-speech-sdk && \
    rm SpeechSDK-Linux.tar.gz && \
    sed -i '/Speech_SegmentationMaximumTimeMs = 9004/d' /opt/azure-speech-sdk/include/c_api/speechapi_c_property_bag.h && \
    rm -rf /tmp/build
