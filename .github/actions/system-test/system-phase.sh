#!/usr/bin/env bash
set -Eeuo pipefail

phase=${1:?usage: system-phase.sh PHASE}
: "${COMPOSE_PROJECT_NAME:?COMPOSE_PROJECT_NAME is required}"
: "${SYSTEM_BUILDER_NAME:?SYSTEM_BUILDER_NAME is required}"
: "${SYSTEM_BASELINE_DIR:?SYSTEM_BASELINE_DIR is required}"
: "${SYSTEM_STATE_DIR:?SYSTEM_STATE_DIR is required}"
: "${SYSTEM_DIAGNOSTICS_DIR:?SYSTEM_DIAGNOSTICS_DIR is required}"

compose=(./tests/system/bin/docker-compose -f docker-compose.yml -f docker-compose.ci.yml)
cache_scope=$(<"${SYSTEM_STATE_DIR}/cache-scope")
diagnostic_work_dir="${SYSTEM_STATE_DIR}/diagnostics-work"
export BUILDX_BUILDER=${SYSTEM_BUILDER_NAME}
export SYSTEM_CACHE_SCOPE=${cache_scope}

case ${phase} in
  validate-contracts)
    test "$(./tests/system/bin/docker-compose version --short)" = "2.24.4"
    scripts/contracts/materialize-baseline.sh "${SYSTEM_BASELINE_DIR}"
    ./tests/system/bin/docker-compose -f docker-compose.yml \
      config --format json > "${SYSTEM_STATE_DIR}/base-compose.json"
    "${compose[@]}" config --format json > "${SYSTEM_STATE_DIR}/system-compose.json"
    ./tests/system/bin/verify-service-image-digests \
      --compose docker-compose.yml \
      --lock tests/system/service-images.lock \
      --baseline "${SYSTEM_BASELINE_DIR}/docker-compose.yml" \
      --forbid-major-change
    go run ./tests/system/cmd/systemcheck compose-contract \
      --base-rendered "${SYSTEM_STATE_DIR}/base-compose.json" \
      --override docker-compose.ci.yml \
      --merged-rendered "${SYSTEM_STATE_DIR}/system-compose.json" \
      --compose-version 2.24.4 \
      --forbid-path docker/ci
    ;;
  validate-native-locks)
    ./bin/check-go-version-consistency
    ./docker/assistant-api/scripts/verify-native-deps.sh \
      docker/assistant-api/native-deps.lock
    ;;
  validate-system-support)
    go test -count=1 ./tests/system/...
    .github/actions/system-test/cleanup_test.sh
    .github/actions/system-test/orchestration_test.sh
    ;;
  create-builder)
    docker buildx create \
      --name "${SYSTEM_BUILDER_NAME}" \
      --driver docker-container \
      --use
    if ! docker buildx inspect "${SYSTEM_BUILDER_NAME}" --bootstrap \
      > "${SYSTEM_STATE_DIR}/builder-bootstrap.log" 2>&1; then
      echo "Buildx builder bootstrap failed; sanitized diagnostics will be collected" >&2
      exit 1
    fi
    ;;
  build-images)
    install -d -m 700 "${diagnostic_work_dir}"
    compose_images_tmp="${diagnostic_work_dir}/compose-images.json.tmp"
    if ! "${compose[@]}" config --format json | jq -cS '
      . as $config
      | ["web-api", "integration-api", "endpoint-api", "assistant-api", "ui", "test-runner"] as $services
      | reduce $services[] as $service ({};
          if ($config.services | has($service)) then
            . + {($service): ($config.services[$service].image // ($config.name + "-" + $service))}
          else
            error("rendered Compose config missing " + $service)
          end
        )
    ' > "${compose_images_tmp}"; then
      rm -f "${compose_images_tmp}"
      echo "System image mapping generation failed" >&2
      exit 1
    fi
    mv "${compose_images_tmp}" "${diagnostic_work_dir}/compose-images.json"
    if ! "${compose[@]}" build --progress plain \
      > "${diagnostic_work_dir}/buildkit.log" 2>&1; then
      echo "System image build failed; sanitized diagnostics will be collected" >&2
      exit 1
    fi
    if ! go run ./tests/system/cmd/systemcheck build-metadata \
      --compose-images "${diagnostic_work_dir}/compose-images.json" \
      --output "${diagnostic_work_dir}/buildkit-metadata.json" \
      > "${SYSTEM_STATE_DIR}/build-metadata-command.log" 2>&1; then
      rm -f "${SYSTEM_STATE_DIR}/build-metadata-command.log"
      echo "System image metadata validation failed" >&2
      exit 1
    fi
    rm -f "${SYSTEM_STATE_DIR}/build-metadata-command.log"
    ;;
  migrations)
    "${compose[@]}" run --rm migrate-web up
    "${compose[@]}" run --rm migrate-integration up
    "${compose[@]}" run --rm migrate-endpoint up
    "${compose[@]}" run --rm migrate-assistant up
    "${compose[@]}" run --rm test-runner \
      systemcheck migrations --require-clean --require-head \
      --report /reports/migrations.json
    ;;
  start-services)
    "${compose[@]}" up -d --no-build \
      postgres redis web-api integration-api endpoint-api assistant-api ui nginx
    ;;
  health)
    "${compose[@]}" run --rm test-runner \
      systemcheck health --timeout-per-service 60s --interval 1s \
      --readiness-key 'PSQL psql://postgres:5432' \
      --reject-arbitrary-true-fallback
    ;;
  ui-nginx)
    "${compose[@]}" run --rm test-runner \
      systemcheck ui-nginx --base-url http://nginx:8080 \
      --require-spa-root --require-hashed-asset \
      --proxy-route /talk_api.TalkService/GetAllAssistantConversation=assistant-api:9007 \
      --proxy-route /web_api.AuthenticationService/ForgotPassword=web-api:9001
    ;;
  assistant-collection-drift)
    python3 openapi/scripts/generate_assistant_postman_collection.py --check
    ;;
  assistant-smoke)
    "${compose[@]}" run --rm test-runner \
      systemcheck assistant-smoke \
      --collection openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json \
      --base-url http://assistant-api:9007 \
      --tmpfs /run/secrets
    ;;
  *)
    echo "Unknown system test phase: ${phase}" >&2
    exit 2
    ;;
esac
