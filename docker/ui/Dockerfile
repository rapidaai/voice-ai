# syntax=docker/dockerfile:1

# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM rapidaai/rapida-node:22-alpine AS builder

WORKDIR /app

COPY ui/package.json ui/yarn.lock ./
RUN --mount=type=cache,target=/root/.cache/yarn \
    yarn config set network-timeout 300000 && \
    yarn install --frozen-lockfile

COPY ui/ ./

# For marketplace: pass --build-arg EDITION=community or EDITION=enterprise
# This swaps the config before building. Default (empty) keeps the dev config.
ARG EDITION
COPY docker/ui/config.*.json /tmp/configs/
RUN if [ -n "$EDITION" ] && [ -f "/tmp/configs/config.${EDITION}.json" ]; then \
      cp "/tmp/configs/config.${EDITION}.json" src/configs/config.development.json; \
    fi

RUN yarn checkTs && yarn build:docker

# ── Stage 2: Serve ──────────────────────────────────────────────────────────
FROM node:22-alpine

RUN npm install -g serve

WORKDIR /app

COPY --from=builder /app/build ./build
COPY docker/ui/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 3000

CMD ["/entrypoint.sh"]
