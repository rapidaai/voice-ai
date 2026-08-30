go_ci_packages := "./api/web-api/... ./api/integration-api/... ./api/endpoint-api/... ./pkg/... ./cmd/web/... ./cmd/integration/... ./cmd/endpoint/... ./config/..."

# Run every required CI stage locally in production order.
ci: ci-doctor ci-basics ci-docker ci-single-service ci-integration ci-smoke

# Run repository, Go, UI, and contract checks.
ci-basics: ci-check ci-repository ci-go ci-ui ci-contracts

# Build every application Docker image.
ci-docker:
    bash just/ci-stack.sh build

# Verify the assistant service with its native runtime dependencies.
ci-single-service: ci-assistant-native

# Validate CI prerequisites without requiring unused host ports.
ci-doctor:
    DOCTOR_SKIP_PORT_CHECK=1 bash just/doctor.sh

# Run CI checks that do not start the full Docker stack.
ci-fast: ci-check ci-go ci-ui ci-contracts

# Validate recipes, workflows, shell scripts, configuration, and generated files.
ci-check:
    just --fmt --check
    go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -color
    bash -n just/*.sh scripts/contracts/*.sh
    bash just/shellcheck.sh
    bash just/ci-service-boundaries.sh
    ./bin/check-go-version-consistency
    bash just/ci-python.sh openapi/scripts/generate_assistant_postman_collection.py --check
    CI_STACK_CACHE_SCOPE=local CI_STACK_AUTH_TOKEN=local CI_STACK_PROJECT_API_KEY=local CI_STACK_REPORTS_DIR="${PWD}/smoke-reports" docker compose -f docker-compose.yml -f docker-compose.ci.yml config --quiet
    PATH="$(bash just/ci-python.sh --bin-dir):$PATH" just validate-development-toolkit

# Run repository policy, commit-message, and filesystem security checks.
ci-repository: ci-license ci-commitlint ci-security

ci-license:
    @test -f LICENSE.md -o -f LICENSE || { echo 'LICENSE file is missing.' >&2; exit 1; }

ci-commitlint:
    bash just/ci-commitlint.sh

ci-security:
    bash just/ci-security.sh

# Run the complete Go validation lane.
ci-go: ci-go-format ci-go-load ci-go-vet ci-go-test ci-go-race ci-go-build ci-go-mod ci-go-lint ci-go-vulnerability

ci-go-format:
    #!/usr/bin/env bash
    unformatted=$(find . -name '*.go' -not -path './protos/*' -not -path './sdks/*' -not -path './vendor/*' -print0 | xargs -0 gofmt -l 2>/dev/null || true)
    if [[ -n "$unformatted" ]]; then
      printf 'The following files are not gofmt-formatted:\n%s\n' "$unformatted" >&2
      exit 1
    fi

ci-go-load:
    CGO_ENABLED=1 go list -deps -test {{go_ci_packages}} >/dev/null

ci-go-vet:
    CGO_ENABLED=0 go vet {{go_ci_packages}}

ci-go-test:
    CGO_ENABLED=1 go test -v -coverprofile=coverage.out -covermode=atomic {{go_ci_packages}}

ci-go-race:
    CGO_ENABLED=1 go test -race -count=1 {{go_ci_packages}}

ci-go-build:
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o bin/web-api ./cmd/web/web.go
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o bin/integration-api ./cmd/integration/integration.go
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o bin/endpoint-api ./cmd/endpoint/endpoint.go

ci-go-mod:
    #!/usr/bin/env bash
    go mod verify
    go mod tidy
    if [[ -n $(git status --porcelain go.mod go.sum) ]]; then
      git diff -- go.mod go.sum
      echo 'go.mod or go.sum is not tidy' >&2
      exit 1
    fi

ci-go-lint:
    bash just/ci-go-lint.sh {{go_ci_packages}}

ci-go-vulnerability:
    CGO_ENABLED=0 go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 {{go_ci_packages}}

# Run the complete UI validation lane.
ci-ui: ci-ui-lint ci-ui-typecheck ci-ui-test ci-ui-build

_ci-ui-install:
    cd ui && yarn install --frozen-lockfile

ci-ui-lint: _ci-ui-install
    cd ui && yarn check:theme-contract
    cd ui && yarn test:theme-contract
    cd ui && yarn test:bundle-size
    cd ui && yarn test:typescript-check
    cd ui && yarn check:public-assets
    cd ui && yarn test:public-assets
    cd ui && yarn lint
    cd ui && yarn lint:css

ci-ui-typecheck: _ci-ui-install
    cd ui && yarn checkTs

ci-ui-test: _ci-ui-install
    cd ui && yarn test --watchAll=false --passWithNoTests --coverage

ci-ui-build: _ci-ui-install
    cd ui && CI=false DISABLE_ESLINT_PLUGIN=true yarn build
    cd ui && yarn check:bundle-size

# Check OpenAPI and protobuf compatibility against a baseline.
ci-contracts baseline="":
    bash just/ci-contracts.sh "{{baseline}}"

# Build and execute the assistant native-runtime CI image.
ci-assistant-native:
    bash just/ci-assistant-native.sh

# Build every service and verify migrations, readiness, routing, and UI delivery.
ci-integration:
    bash just/ci-stack.sh integration

# Build every service and run REST and SDK smoke tests with all supported authentication methods.
ci-smoke:
    bash just/ci-stack.sh smoke
