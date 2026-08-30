compose := "env DOCKER_BUILDKIT=1 COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_DEFAULT_PLATFORM=linux/amd64 docker compose -f docker-compose.yml"
compose_knowledge := compose + " -f docker-compose.knowledge.yml"
core_services := "ui web-api integration-api endpoint-api assistant-api"
knowledge_services := core_services + " document-api"

# Create local data directories and permissions.
setup-local:
    @echo "Creating local data directories..."
    mkdir -p "${HOME}/rapida-data/assets/opensearch" "${HOME}/rapida-data/assets/db" "${HOME}/rapida-data/assets/redis"
    chmod u+rwx "${HOME}/rapida-data" "${HOME}/rapida-data/assets" "${HOME}/rapida-data/assets/opensearch" "${HOME}/rapida-data/assets/db" "${HOME}/rapida-data/assets/redis" 2>/dev/null || true
    @echo "Setup complete. Run 'just up-all'."

# Validate Docker, disk, directories, and ports.
doctor:
    bash just/doctor.sh

# Start all core services using sequential builds.
up-all: up-all-safe

# Start all core services using parallel builds.
up-all-fast: doctor
    @echo "Starting all core services..."
    {{compose}} up -d
    @just status

# Start all core services using low-memory sequential builds.
up-all-safe: doctor
    @echo "Building core services sequentially..."
    @for service in {{core_services}}; do echo "  -> building $service"; {{compose}} build "$service"; done
    {{compose}} up -d --no-build
    @just status

# Start all services including the knowledge stack.
up-all-with-knowledge: up-all-with-knowledge-safe

# Start all services including knowledge using parallel builds.
up-all-with-knowledge-fast: doctor
    @echo "Starting all services including knowledge..."
    {{compose_knowledge}} up -d
    @just status

# Start all services including knowledge using sequential builds.
up-all-with-knowledge-safe: doctor
    @echo "Building all services sequentially, including knowledge..."
    @for service in {{knowledge_services}}; do echo "  -> building $service"; {{compose_knowledge}} build "$service"; done
    {{compose_knowledge}} up -d --no-build
    @just status

# Start one Compose service.
up-service service:
    {{compose}} up -d "{{service}}"

up-ui: (up-service "ui")
up-web: (up-service "web-api")
up-integration: (up-service "integration-api")
up-endpoint: (up-service "endpoint-api")
up-assistant: (up-service "assistant-api")
up-db: (up-service "postgres")
up-nginx: (up-service "nginx")
up-redis: (up-service "redis")

up-document:
    {{compose_knowledge}} up -d document-api

up-opensearch:
    {{compose_knowledge}} up -d opensearch

alias up := up-all

# Stop all services and optional knowledge services.
down-all:
    {{compose_knowledge}} down

# Stop one Compose service.
down-service service:
    {{compose}} stop "{{service}}"

down-ui: (down-service "ui")
down-web: (down-service "web-api")
down-integration: (down-service "integration-api")
down-endpoint: (down-service "endpoint-api")
down-assistant: (down-service "assistant-api")
down-db: (down-service "postgres")
down-nginx: (down-service "nginx")
down-redis: (down-service "redis")

down-document:
    {{compose_knowledge}} stop document-api

down-opensearch:
    {{compose_knowledge}} stop opensearch

alias down := down-all

# Build all core services sequentially.
build-all: build-all-safe

# Build all core services in parallel.
build-all-fast: doctor
    {{compose}} build {{core_services}}

# Build all core services sequentially.
build-all-safe: doctor
    @for service in {{core_services}}; do echo "  -> building $service"; {{compose}} build "$service"; done

# Build all services including knowledge sequentially.
build-all-with-knowledge: build-all-with-knowledge-safe

# Build all services including knowledge in parallel.
build-all-with-knowledge-fast: doctor
    {{compose_knowledge}} build {{knowledge_services}}

# Build all services including knowledge sequentially.
build-all-with-knowledge-safe: doctor
    @for service in {{knowledge_services}}; do echo "  -> building $service"; {{compose_knowledge}} build "$service"; done

# Build one core Compose service.
build-service service:
    {{compose}} build "{{service}}"

build-ui: (build-service "ui")
build-web: (build-service "web-api")
build-integration: (build-service "integration-api")
build-endpoint: (build-service "endpoint-api")
build-assistant: (build-service "assistant-api")

build-document:
    {{compose_knowledge}} build document-api

alias build := build-web

# Rebuild all core services without cache.
rebuild-all: doctor
    {{compose}} build --no-cache {{core_services}}

# Rebuild all services including knowledge without cache.
rebuild-all-with-knowledge: doctor
    {{compose_knowledge}} build --no-cache {{knowledge_services}}

# Rebuild one core Compose service without cache.
rebuild-service service:
    {{compose}} build --no-cache "{{service}}"

rebuild-ui: (rebuild-service "ui")
rebuild-web: (rebuild-service "web-api")
rebuild-integration: (rebuild-service "integration-api")
rebuild-endpoint: (rebuild-service "endpoint-api")
rebuild-assistant: (rebuild-service "assistant-api")
rebuild-nginx: (rebuild-service "nginx")

rebuild-document:
    {{compose_knowledge}} build --no-cache document-api

alias rebuild := rebuild-web

# Follow logs for all services.
logs-all:
    {{compose_knowledge}} logs -f

# Follow logs for one core Compose service.
logs-service service:
    {{compose}} logs -f "{{service}}"

logs-ui: (logs-service "ui")
logs-web: (logs-service "web-api")
logs-integration: (logs-service "integration-api")
logs-endpoint: (logs-service "endpoint-api")
logs-assistant: (logs-service "assistant-api")
logs-db: (logs-service "postgres")
logs-redis: (logs-service "redis")

logs-document:
    {{compose_knowledge}} logs -f document-api

logs-opensearch:
    {{compose_knowledge}} logs -f opensearch

alias logs := logs-all

# Restart all services.
restart-all:
    {{compose_knowledge}} restart

# Restart one core Compose service.
restart-service service:
    {{compose}} restart "{{service}}"

restart-ui: (restart-service "ui")
restart-web: (restart-service "web-api")
restart-integration: (restart-service "integration-api")
restart-endpoint: (restart-service "endpoint-api")
restart-assistant: (restart-service "assistant-api")
restart-nginx: (restart-service "nginx")

restart-document:
    {{compose_knowledge}} restart document-api

alias restart := restart-all

# Show service status and local endpoints.
status:
    {{compose}} ps
    @printf '%s\n' \
      'UI:              http://localhost:3000' \
      'API Gateway:     http://localhost:8080' \
      'Integration API: http://localhost:9004' \
      'Endpoint API:    http://localhost:9005' \
      'Assistant API:   http://localhost:9007' \
      'SIP:             udp://localhost:5090'

alias ps := status
alias ps-all := status

# Open a shell in one core Compose service.
shell-service service command="sh":
    {{compose}} exec "{{service}}" {{command}}

shell-ui: (shell-service "ui")
shell-nginx: (shell-service "nginx")
shell-assistant: (shell-service "assistant-api")
shell-web: (shell-service "web-api")
shell-integration: (shell-service "integration-api")
shell-endpoint: (shell-service "endpoint-api")

shell-document:
    {{compose_knowledge}} exec document-api sh

shell-db:
    {{compose}} exec postgres psql -U rapida_user -d web_db

alias shell := shell-web
alias db-shell := shell-db

# Remove Compose volumes.
clean-volumes:
    {{compose_knowledge}} down -v

# Remove Compose resources and locally built service images.
clean:
    {{compose_knowledge}} down -v
    image_ids="$(docker images --format '{{"{{"}}.Repository{{"}}"}} {{"{{"}}.ID{{"}}"}}' | awk '$1 ~ /(web-api|integration-api|endpoint-api|assistant-api|document-api|ui)/ {print $2}' | sort -u)"; if [[ -n "$image_ids" ]]; then while IFS= read -r image_id; do docker rmi "$image_id" || true; done <<< "$image_ids"; fi

# Start PostgreSQL and Redis.
deps:
    {{compose}} up -d postgres redis

# Start PostgreSQL, Redis, and OpenSearch.
deps-knowledge:
    {{compose_knowledge}} up -d postgres redis opensearch

# Build and start the core stack.
full: up-all

# Rebuild, start, and follow the core stack.
dev: rebuild-all
    {{compose}} up -d --no-build
    {{compose}} logs -f
