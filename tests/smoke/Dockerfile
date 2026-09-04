# syntax=docker/dockerfile:1
FROM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32

RUN apk add --no-cache ca-certificates curl jq postgresql-client redis

WORKDIR /workspace
COPY tests/smoke/package.json tests/smoke/package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force

COPY tests/smoke/run.sh /usr/local/bin/run-ci-tests
COPY tests/smoke/sdk-smoke.js /workspace/sdk-smoke.js
COPY tests/integration/telephony /workspace/tests/integration/telephony
COPY openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json /workspace/openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json

RUN chmod 0755 /usr/local/bin/run-ci-tests && \
    find /workspace/tests/integration/telephony -name run.sh -exec chmod 0755 {} + && \
    mkdir -p /reports && \
    chown node:node /reports

USER node
ENTRYPOINT ["run-ci-tests"]
