.PHONY: help up down build rebuild logs clean restart ps shell db-shell \
        up-all up-all-with-knowledge up-web up-integration up-endpoint up-db up-redis up-opensearch deps deps-knowledge \
        up-all-safe up-all-with-knowledge-safe \
        up-all-fast up-all-with-knowledge-fast \
        down-all down-web down-integration down-endpoint down-db down-redis down-opensearch \
        build-all build-all-with-knowledge build-web build-integration build-endpoint \
        build-all-safe build-all-with-knowledge-safe \
        build-all-fast build-all-with-knowledge-fast \
        rebuild-all rebuild-all-with-knowledge rebuild-web rebuild-integration rebuild-endpoint \
        logs-all logs-web logs-integration logs-endpoint logs-db logs-redis logs-opensearch \
        restart-all restart-web restart-integration restart-endpoint \
        ps-all shell-web shell-integration shell-endpoint db-shell \
        push-base-images \
        push-rapida-golang-bookworm push-rapida-golang-alpine push-rapida-alpine \
        push-rapida-debian-slim push-rapida-node-alpine push-rapida-python \
        test-tts-integration test-stt-integration test-transformer-integration \
        validate-development-process validate-agent-tooling validate-rfc-layout validate-development-toolkit agent-finalize sign-approved-plan orca-development-run orca-rfc-release orca-confirm-rfc-create orca-confirm-rfc-collect orca-panel orca-panel-open doctor

COMPOSE           := DOCKER_BUILDKIT=1 COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_DEFAULT_PLATFORM=linux/amd64 docker compose -f docker-compose.yml
COMPOSE_KNOWLEDGE := DOCKER_BUILDKIT=1 COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_DEFAULT_PLATFORM=linux/amd64 docker compose -f docker-compose.yml -f docker-compose.knowledge.yml

help:
	@echo ""
	@echo "╔════════════════════════════════════════════════════════════════╗"
	@echo "║          Docker Compose Service Management                    ║"
	@echo "╚════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "STARTUP COMMANDS:"
	@echo "  make up-all                    - Start all services (sequential low-memory build; default)"
	@echo "  make up-all-safe               - Start all services with sequential low-memory build"
	@echo "  make up-all-fast               - Start all services with parallel build"
	@echo "  make up-all-with-knowledge     - Start all services including opensearch and document-api"
	@echo "  make up-all-with-knowledge-safe- Start all services + knowledge with sequential low-memory build"
	@echo "  make up-all-with-knowledge-fast- Start all services + knowledge with parallel build"
	@echo "  make up-web                    - Start web-api only"
	@echo "  make up-integration            - Start integration-api only"
	@echo "  make up-endpoint               - Start endpoint-api only"
	@echo "  make up-db                     - Start PostgreSQL only"
	@echo "  make up-redis                  - Start Redis only"
	@echo "  make up-opensearch             - Start OpenSearch only"
	@echo "  make up-nginx                  - Start nginx only"
	@echo ""
	@echo "KNOWLEDGE BASE (OpenSearch + document-api):"
	@echo "  make up-all-with-knowledge     - Start all services including knowledge"
	@echo "  make build-all-with-knowledge  - Build all images including document-api"
	@echo "  make rebuild-all-with-knowledge- Rebuild all including document-api (no cache)"
	@echo "  Note: docker-compose.knowledge.yml switches assistant-api to docker/assistant-api/assistant.knowledge.yml"
	@echo ""
	@echo "SHUTDOWN COMMANDS:"
	@echo "  make down-all            - Stop all services"
	@echo "  make down-web            - Stop web-api only"
	@echo "  make down-integration    - Stop integration-api only"
	@echo "  make down-endpoint       - Stop endpoint-api only"
	@echo "  make down-db             - Stop PostgreSQL only"
	@echo "  make down-redis          - Stop Redis only"
	@echo "  make down-opensearch     - Stop OpenSearch only"
	@echo "  make down-nginx       	  - Stop nginx only"
	@echo ""
	@echo "BUILD COMMANDS:"
	@echo "  make build-all                 - Build all services sequentially (default)"
	@echo "  make build-all-safe            - Build all services sequentially (low-memory)"
	@echo "  make build-all-fast            - Build all services in parallel"
	@echo "  make push-base-images          - Build + push all rapida-* base images (run when versions change)"
	@echo "  make push-rapida-golang-bookworm - Rebuild + push rapidaai/rapida-golang:1.25.13-bookworm"
	@echo "  make push-rapida-golang-alpine   - Rebuild + push rapidaai/rapida-golang:1.25.13-alpine"
	@echo "  make push-rapida-alpine          - Rebuild + push rapidaai/rapida-alpine:3.21"
	@echo "  make push-rapida-debian-slim     - Rebuild + push rapidaai/rapida-debian:bookworm-slim"
	@echo "  make push-rapida-node-alpine     - Rebuild + push rapidaai/rapida-node:22-alpine"
	@echo "  make push-rapida-python          - Rebuild + push rapidaai/rapida-python:3.11"
	@echo "  make build-all-with-knowledge  - Build all services including document-api"
	@echo "  make build-all-with-knowledge-safe - Build all services + knowledge sequentially (low-memory)"
	@echo "  make build-all-with-knowledge-fast - Build all services + knowledge in parallel"
	@echo "  make build-web                 - Build web-api image"
	@echo "  make build-integration         - Build integration-api image"
	@echo "  make build-endpoint            - Build endpoint-api image"
	@echo "  make rebuild-all               - Rebuild all services (no cache, no document-api)"
	@echo "  make rebuild-all-with-knowledge- Rebuild all including document-api (no cache)"
	@echo "  make rebuild-web               - Rebuild web-api (no cache)"
	@echo "  make rebuild-integration       - Rebuild integration-api (no cache)"
	@echo "  make rebuild-endpoint          - Rebuild endpoint-api (no cache)"
	@echo ""
	@echo "MONITORING COMMANDS:"
	@echo "  make logs-all            - View all service logs"
	@echo "  make logs-web            - View web-api logs"
	@echo "  make logs-integration    - View integration-api logs"
	@echo "  make logs-endpoint       - View endpoint-api logs"
	@echo "  make logs-db             - View PostgreSQL logs"
	@echo "  make logs-redis          - View Redis logs"
	@echo "  make logs-opensearch     - View OpenSearch logs"
	@echo "  make ps-all              - Show all running containers"
	@echo ""
	@echo "RESTART COMMANDS:"
	@echo "  make restart-all         - Restart all services"
	@echo "  make restart-web         - Restart web-api"
	@echo "  make restart-integration - Restart integration-api"
	@echo "  make restart-endpoint    - Restart endpoint-api"
	@echo ""
	@echo "SHELL COMMANDS:"
	@echo "  make shell-web           - Open web-api container shell"
	@echo "  make shell-integration   - Open integration-api container shell"
	@echo "  make shell-endpoint      - Open endpoint-api container shell"
	@echo "  make shell-db           - Open PostgreSQL shell"
	@echo ""
	@echo "MAINTENANCE:"
	@echo "  make clean               - Stop and remove containers, volumes, images"
	@echo "  make clean-volumes       - Remove only volumes"
	@echo "  make status              - Show container status and ports"
	@echo "  make doctor              - Run preflight checks before build/start"
	@echo ""
	@echo "SETUP:"
	@echo "  make setup-local         - Create directories and set local permissions"
	@echo ""

# ============================================================================
# STARTUP TARGETS - Individual Services
# ============================================================================

setup-local:
	@echo "Creating local data directories..."
	mkdir -p ${HOME}/rapida-data/assets/opensearch
	mkdir -p ${HOME}/rapida-data/assets/db
	mkdir -p ${HOME}/rapida-data/assets/redis
	@echo "Applying best-effort directory permissions..."
	chmod u+rwx ${HOME}/rapida-data ${HOME}/rapida-data/assets \
		${HOME}/rapida-data/assets/opensearch ${HOME}/rapida-data/assets/db ${HOME}/rapida-data/assets/redis 2>/dev/null || true
	@echo "✓ Setup complete. You can now run 'make up-all'"

doctor:
	@echo "Running Docker preflight checks..."
	@set -e; \
	errors=0; \
	if ! command -v docker >/dev/null 2>&1; then \
		echo "✗ docker CLI not found. Install Docker Desktop / Docker Engine."; \
		errors=$$((errors + 1)); \
	else \
		echo "✓ docker CLI found"; \
	fi; \
	if ! docker info >/dev/null 2>&1; then \
		echo "✗ Docker daemon is not reachable. Start Docker and retry."; \
		errors=$$((errors + 1)); \
	else \
		echo "✓ docker daemon reachable"; \
	fi; \
	if ! docker compose version >/dev/null 2>&1; then \
		echo "✗ docker compose plugin not found."; \
		errors=$$((errors + 1)); \
	else \
		echo "✓ docker compose available"; \
	fi; \
	free_kb=$$(df -Pk "$$HOME" | awk 'NR==2 {print $$4}'); \
	min_kb=$$((20 * 1024 * 1024)); \
	if [ -z "$$free_kb" ]; then \
		echo "✗ Unable to detect free disk space under $$HOME."; \
		errors=$$((errors + 1)); \
	elif [ "$$free_kb" -lt "$$min_kb" ]; then \
		free_gb=$$(awk "BEGIN {printf \"%.1f\", $$free_kb/1024/1024}"); \
		echo "✗ Low disk space: $${free_gb}GB free, at least 20GB required for first Docker build."; \
		errors=$$((errors + 1)); \
	else \
		free_gb=$$(awk "BEGIN {printf \"%.1f\", $$free_kb/1024/1024}"); \
		echo "✓ disk space looks good ($${free_gb}GB free)"; \
	fi; \
	if [ "$${DOCTOR_SKIP_CACHE_CHECK:-0}" != "1" ]; then \
		build_cache_mb=$$(docker system df --format '{{.Type}} {{.Size}}' 2>/dev/null | awk '$$1=="Build" && $$2=="Cache" {size=$$3} END { \
			if (size ~ /kB$$/) {sub(/kB$$/, "", size); mb=size/1024} \
			else if (size ~ /MB$$/) {sub(/MB$$/, "", size); mb=size} \
			else if (size ~ /GB$$/) {sub(/GB$$/, "", size); mb=size*1024} \
			else if (size ~ /TB$$/) {sub(/TB$$/, "", size); mb=size*1024*1024} \
			else if (size ~ /B$$/) {sub(/B$$/, "", size); mb=size/(1024*1024)} \
			else {mb=-1} \
			if (mb >= 0) printf "%.0f", mb; \
		}'); \
		if [ -n "$$build_cache_mb" ] && [ "$$build_cache_mb" -gt 12288 ]; then \
			build_cache_gb=$$(awk "BEGIN {printf \"%.1f\", $$build_cache_mb/1024}"); \
			echo "! Build cache is large (~$${build_cache_gb}GB)."; \
			echo "  Run: docker builder prune -af"; \
			echo "  (Set DOCTOR_SKIP_CACHE_CHECK=1 to bypass this check.)"; \
		else \
			echo "✓ build cache usage is within preflight limit"; \
		fi; \
	fi; \
	for d in "$$HOME/rapida-data/assets" "$$HOME/rapida-data/assets/db" "$$HOME/rapida-data/assets/redis"; do \
		mkdir -p "$$d"; \
		if [ ! -w "$$d" ]; then \
			echo "✗ Directory is not writable: $$d"; \
			errors=$$((errors + 1)); \
		else \
			touch "$$d/.doctor-write-test" 2>/dev/null || true; \
			rm -f "$$d/.doctor-write-test" 2>/dev/null || true; \
			echo "✓ writable directory: $$d"; \
		fi; \
	done; \
	if command -v lsof >/dev/null 2>&1; then \
		for p in 3000 8080 9004 9005 9007 4573; do \
			owner=$$(lsof -nP -iTCP:$$p -sTCP:LISTEN 2>/dev/null | awk 'NR==2 {print $$1}'); \
			if [ -z "$$owner" ]; then \
				echo "✓ TCP port $$p is free"; \
			elif echo "$$owner" | grep -Eq '^(com\.docke|docker-proxy|docker)$$'; then \
				echo "! TCP port $$p is in use by Docker ($$owner); continuing."; \
			else \
				echo "✗ TCP port $$p is already in use by $$owner."; \
				errors=$$((errors + 1)); \
			fi; \
		done; \
		udp_owner=$$(lsof -nP -iUDP:5090 2>/dev/null | awk 'NR==2 {print $$1}'); \
		if [ -z "$$udp_owner" ]; then \
			echo "✓ UDP port 5090 is free"; \
		elif echo "$$udp_owner" | grep -Eq '^(com\.docke|docker-proxy|docker)$$'; then \
			echo "! UDP port 5090 is in use by Docker ($$udp_owner); continuing."; \
		else \
			echo "✗ UDP port 5090 is already in use by $$udp_owner."; \
			errors=$$((errors + 1)); \
		fi; \
	else \
		echo "! lsof not found, skipping port checks."; \
	fi; \
	if [ $$errors -gt 0 ]; then \
		echo ""; \
		echo "Preflight failed with $$errors issue(s). Fix the above and rerun."; \
		exit 1; \
	fi; \
	echo "✓ Preflight checks passed."


up-all:
	@$(MAKE) up-all-safe

up-all-fast:
	@$(MAKE) doctor
	@echo "Starting all services (without opensearch/document-api)..."
	$(COMPOSE) up -d
	@echo "✓ All services started"
	@echo "  Run 'make up-all-with-knowledge' to include opensearch + document-api"
	@echo "  If your machine is memory-constrained, run 'make up-all-safe'"
	@$(MAKE) status

up-all-safe:
	@$(MAKE) doctor
	@echo "Building all services sequentially (low-memory mode)..."
	@for service in ui web-api integration-api endpoint-api assistant-api; do \
		echo "  -> building $$service"; \
		$(COMPOSE) build $$service || exit 1; \
	done
	@echo "Starting all services (without opensearch/document-api)..."
	$(COMPOSE) up -d --no-build
	@echo "✓ All services started (low-memory mode)"
	@echo "  Run 'make up-all-with-knowledge-safe' to include opensearch + document-api"
	@$(MAKE) status

up-all-with-knowledge:
	@$(MAKE) up-all-with-knowledge-safe

up-all-with-knowledge-fast:
	@$(MAKE) doctor
	@echo "Starting all services including knowledge base..."
	$(COMPOSE_KNOWLEDGE) up -d
	@echo "✓ All services started (with knowledge base)"
	@echo "  If your machine is memory-constrained, run 'make up-all-with-knowledge-safe'"
	@$(MAKE) status

up-all-with-knowledge-safe:
	@$(MAKE) doctor
	@echo "Building all services sequentially (low-memory mode, with knowledge)..."
	@for service in ui web-api integration-api endpoint-api assistant-api document-api; do \
		echo "  -> building $$service"; \
		$(COMPOSE_KNOWLEDGE) build $$service || exit 1; \
	done
	@echo "Starting all services including knowledge base..."
	$(COMPOSE_KNOWLEDGE) up -d --no-build
	@echo "✓ All services started (low-memory mode, with knowledge base)"
	@$(MAKE) status

up-ui:
	@echo "Starting ui..."
	$(COMPOSE) up -d ui
	@echo "✓ ui started on port 3000"

up-document:
	@echo "Starting document-api..."
	$(COMPOSE_KNOWLEDGE) up -d document-api
	@echo "✓ document-api started on port 9010"

up-web:
	@echo "Starting web-api..."
	$(COMPOSE) up -d web-api
	@echo "✓ web-api started on port 9001"

up-integration:
	@echo "Starting integration-api..."
	$(COMPOSE) up -d integration-api
	@echo "✓ integration-api started on port 9004"

up-endpoint:
	@echo "Starting endpoint-api..."
	$(COMPOSE) up -d endpoint-api
	@echo "✓ endpoint-api started on port 9005"

up-assistant:
	@echo "Starting assistant-api..."
	$(COMPOSE) up -d assistant-api
	@echo "✓ assistant-api started on port 9007"

up-db:
	@echo "Starting PostgreSQL..."
	$(COMPOSE) up -d postgres
	@echo "✓ PostgreSQL started on port 5432"

up-nginx:
	@echo "Starting nginx..."
	$(COMPOSE) up -d nginx
	@echo "✓ nginx started on port 8080"

up-redis:
	@echo "Starting Redis..."
	$(COMPOSE) up -d redis
	@echo "✓ Redis started on port 6379"

up-opensearch:
	@echo "Starting OpenSearch..."
	$(COMPOSE_KNOWLEDGE) up -d opensearch
	@echo "✓ OpenSearch started on port 9200"

# Legacy aliases
up: up-all

# ============================================================================
# SHUTDOWN TARGETS - Individual Services
# ============================================================================

down-all:
	@echo "Stopping all services..."
	$(COMPOSE_KNOWLEDGE) down
	@echo "✓ All services stopped"

down-ui:
	@echo "Stopping ui..."
	$(COMPOSE) stop ui
	@echo "✓ ui stopped"

down-web:
	@echo "Stopping web-api..."
	$(COMPOSE) stop web-api
	@echo "✓ web-api stopped"

down-document:
	@echo "Stopping document-api..."
	$(COMPOSE_KNOWLEDGE) stop document-api
	@echo "✓ document-api stopped"

down-assistant:
	@echo "Stopping assistant-api..."
	$(COMPOSE) stop assistant-api
	@echo "✓ assistant-api stopped"

down-integration:
	@echo "Stopping integration-api..."
	$(COMPOSE) stop integration-api
	@echo "✓ integration-api stopped"

down-endpoint:
	@echo "Stopping endpoint-api..."
	$(COMPOSE) stop endpoint-api
	@echo "✓ endpoint-api stopped"

down-db:
	@echo "Stopping PostgreSQL..."
	$(COMPOSE) stop postgres
	@echo "✓ PostgreSQL stopped"

down-redis:
	@echo "Stopping Redis..."
	$(COMPOSE) stop redis
	@echo "✓ Redis stopped"

down-nginx:
	@echo "Stopping nginx..."
	$(COMPOSE) stop nginx
	@echo "✓ nginx stopped"

down-opensearch:
	@echo "Stopping OpenSearch..."
	$(COMPOSE_KNOWLEDGE) stop opensearch
	@echo "✓ OpenSearch stopped"

# Legacy alias
down: down-all

# ============================================================================
# BUILD TARGETS
# ============================================================================

push-rapida-golang-bookworm:
	@echo "Building rapidaai/rapida-golang:1.25.13-bookworm..."
	DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-golang-bookworm.Dockerfile -t rapidaai/rapida-golang:1.25.13-bookworm .
	docker push rapidaai/rapida-golang:1.25.13-bookworm
	@echo "✓ rapidaai/rapida-golang:1.25.13-bookworm pushed"

push-rapida-golang-alpine:
	@echo "Building rapidaai/rapida-golang:1.25.13-alpine..."
	DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-golang-alpine.Dockerfile -t rapidaai/rapida-golang:1.25.13-alpine .
	docker push rapidaai/rapida-golang:1.25.13-alpine
	@echo "✓ rapidaai/rapida-golang:1.25.13-alpine pushed"

push-rapida-alpine:
	@echo "Building rapidaai/rapida-alpine:3.21..."
	DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-alpine.Dockerfile -t rapidaai/rapida-alpine:3.21 .
	docker push rapidaai/rapida-alpine:3.21
	@echo "✓ rapidaai/rapida-alpine:3.21 pushed"

push-rapida-debian-slim:
	@echo "Building rapidaai/rapida-debian:bookworm-slim..."
	DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-debian-slim.Dockerfile -t rapidaai/rapida-debian:bookworm-slim .
	docker push rapidaai/rapida-debian:bookworm-slim
	@echo "✓ rapidaai/rapida-debian:bookworm-slim pushed"

push-rapida-node-alpine:
	@echo "Building rapidaai/rapida-node:22-alpine..."
	DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-node-alpine.Dockerfile -t rapidaai/rapida-node:22-alpine .
	docker push rapidaai/rapida-node:22-alpine
	@echo "✓ rapidaai/rapida-node:22-alpine pushed"

push-rapida-python:
	@echo "Building rapidaai/rapida-python:3.11..."
	DOCKER_BUILDKIT=1 docker build -f docker/base/rapida-python.Dockerfile -t rapidaai/rapida-python:3.11 .
	docker push rapidaai/rapida-python:3.11
	@echo "✓ rapidaai/rapida-python:3.11 pushed"

push-base-images: push-rapida-golang-bookworm push-rapida-golang-alpine push-rapida-alpine push-rapida-debian-slim push-rapida-node-alpine push-rapida-python
	@echo "✓ All base images pushed to Docker Hub"

build-all:
	@$(MAKE) build-all-safe

build-all-fast:
	@$(MAKE) doctor
	@echo "Building all services (without document-api/opensearch)..."
	$(COMPOSE) build ui web-api integration-api endpoint-api assistant-api
	@echo "✓ All services built"

build-all-safe:
	@$(MAKE) doctor
	@echo "Building all services sequentially (low-memory mode)..."
	@for service in ui web-api integration-api endpoint-api assistant-api; do \
		echo "  -> building $$service"; \
		$(COMPOSE) build $$service || exit 1; \
	done
	@echo "✓ All services built (low-memory mode)"

build-all-with-knowledge:
	@$(MAKE) build-all-with-knowledge-safe

build-all-with-knowledge-fast:
	@$(MAKE) doctor
	@echo "Building all services including document-api..."
	$(COMPOSE_KNOWLEDGE) build ui web-api integration-api endpoint-api assistant-api document-api
	@echo "✓ All services built (with knowledge base)"

build-all-with-knowledge-safe:
	@$(MAKE) doctor
	@echo "Building all services including document-api sequentially (low-memory mode)..."
	@for service in ui web-api integration-api endpoint-api assistant-api document-api; do \
		echo "  -> building $$service"; \
		$(COMPOSE_KNOWLEDGE) build $$service || exit 1; \
	done
	@echo "✓ All services built (low-memory mode, with knowledge base)"

build-ui:
	@echo "Building ui..."
	$(COMPOSE) build ui
	@echo "✓ ui built"

build-web:
	@echo "Building web-api..."
	$(COMPOSE) build web-api
	@echo "✓ web-api built"

build-document:
	@echo "Building document-api..."
	$(COMPOSE_KNOWLEDGE) build document-api
	@echo "✓ document-api built"

build-assistant:
	@echo "Building assistant-api..."
	$(COMPOSE) build assistant-api
	@echo "✓ assistant-api built"

build-integration:
	@echo "Building integration-api..."
	$(COMPOSE) build integration-api
	@echo "✓ integration-api built"

build-endpoint:
	@echo "Building endpoint-api..."
	$(COMPOSE) build endpoint-api
	@echo "✓ endpoint-api built"

rebuild-all:
	@$(MAKE) doctor
	@echo "Rebuilding all services (no cache, without document-api/opensearch)..."
	$(COMPOSE) build --no-cache ui web-api integration-api endpoint-api assistant-api
	@echo "✓ All services rebuilt"

rebuild-all-with-knowledge:
	@$(MAKE) doctor
	@echo "Rebuilding all services including document-api (no cache)..."
	$(COMPOSE_KNOWLEDGE) build --no-cache ui web-api integration-api endpoint-api assistant-api document-api
	@echo "✓ All services rebuilt (with knowledge base)"

rebuild-web:
	@echo "Rebuilding web-api (no cache)..."
	$(COMPOSE) build --no-cache web-api
	@echo "✓ web-api rebuilt"

rebuild-nginx:
	@echo "Rebuilding nginx (no cache)..."
	$(COMPOSE) build --no-cache nginx
	@echo "✓ nginx rebuilt"
	
rebuild-document:
	@echo "Rebuilding document-api (no cache)..."
	$(COMPOSE_KNOWLEDGE) build --no-cache document-api
	@echo "✓ document-api rebuilt"


rebuild-assistant:
	@echo "Rebuilding assistant-api (no cache)..."
	$(COMPOSE) build --no-cache assistant-api
	@echo "✓ assistant-api rebuilt"

rebuild-ui:
	@echo "Rebuilding ui (no cache)..."
	$(COMPOSE) build --no-cache ui
	@echo "✓ ui rebuilt"

rebuild-integration:
	@echo "Rebuilding integration-api (no cache)..."
	$(COMPOSE) build --no-cache integration-api
	@echo "✓ integration-api rebuilt"

rebuild-endpoint:
	@echo "Rebuilding endpoint-api (no cache)..."
	$(COMPOSE) build --no-cache endpoint-api
	@echo "✓ endpoint-api rebuilt"

# Legacy aliases
build: build-web
rebuild: rebuild-web

# ============================================================================
# LOGGING TARGETS
# ============================================================================

logs-all:
	$(COMPOSE_KNOWLEDGE) logs -f

logs-ui:
	$(COMPOSE) logs -f ui

logs-web:
	$(COMPOSE) logs -f web-api


logs-document:
	$(COMPOSE_KNOWLEDGE) logs -f document-api


logs-assistant:
	$(COMPOSE) logs -f assistant-api

logs-integration:
	$(COMPOSE) logs -f integration-api

logs-endpoint:
	$(COMPOSE) logs -f endpoint-api

logs-db:
	$(COMPOSE) logs -f postgres

logs-redis:
	$(COMPOSE) logs -f redis

logs-opensearch:
	$(COMPOSE_KNOWLEDGE) logs -f opensearch

# Legacy alias
logs: logs-all

# ============================================================================
# RESTART TARGETS
# ============================================================================

restart-all:
	@echo "Restarting all services..."
	$(COMPOSE_KNOWLEDGE) restart
	@echo "✓ All services restarted"

restart-nginx:
	@echo "Restarting nginx..."
	$(COMPOSE) restart nginx
	@echo "✓ nginx restarted"

restart-ui:
	@echo "Restarting ui..."
	$(COMPOSE) restart ui
	@echo "✓ ui restarted"

restart-web:
	@echo "Restarting web-api..."
	$(COMPOSE) restart web-api
	@echo "✓ web-api restarted"

restart-document:
	@echo "Restarting document-api..."
	$(COMPOSE_KNOWLEDGE) restart document-api
	@echo "✓ document-api restarted"


restart-assistant:
	@echo "Restarting assistant-api..."
	$(COMPOSE) restart assistant-api
	@echo "✓ assistant-api restarted"

restart-integration:
	@echo "Restarting integration-api..."
	$(COMPOSE) restart integration-api
	@echo "✓ integration-api restarted"

restart-endpoint:
	@echo "Restarting endpoint-api..."
	$(COMPOSE) restart endpoint-api
	@echo "✓ endpoint-api restarted"

# Legacy alias
restart: restart-all

# ============================================================================
# STATUS TARGETS
# ============================================================================

ps-all:
	@echo ""
	@echo "Running Containers:"
	@echo "==================="
	$(COMPOSE) ps
	@echo ""

status: ps-all
	@echo "Service Ports:"
	@echo "=============="
	@echo "  UI:               http://localhost:3000"
	@echo "  API Gateway:      http://localhost:8080"
	@echo "  Web-API:          internal only (no host port)"
	@echo "  Integration-API:  http://localhost:9004"
	@echo "  Endpoint-API:     http://localhost:9005"
	@echo "  Assistant-API:    http://localhost:9007"
	@echo "  SIP:              udp://localhost:5090"
	@echo "  PostgreSQL:       internal only (no host port)"
	@echo "  Redis:            internal only (no host port)"
	@echo "  OpenSearch:       internal only (run make up-all-with-knowledge)"
	@echo "  Document-API:     internal only (run make up-all-with-knowledge)"
	@echo ""

ps: ps-all

# ============================================================================
# SHELL/ACCESS TARGETS
# ============================================================================

shell-ui:
	$(COMPOSE) exec ui sh

shell-nginx:
	$(COMPOSE) exec nginx sh

shell-assistant:
	$(COMPOSE) exec assistant-api sh

shell-document:
	$(COMPOSE_KNOWLEDGE) exec document-api sh

shell-web:
	$(COMPOSE) exec web-api sh

shell-integration:
	$(COMPOSE) exec integration-api sh

shell-endpoint:
	$(COMPOSE) exec endpoint-api sh

shell-db:
	$(COMPOSE) exec postgres psql -U rapida_user -d web_db

# Legacy alias
shell: shell-web

# ============================================================================
# MAINTENANCE TARGETS
# ============================================================================

clean-volumes:
	@echo "Removing volumes..."
	$(COMPOSE_KNOWLEDGE) down -v
	@echo "✓ Volumes removed"

clean:
	@echo "Cleaning up Docker resources..."
	$(COMPOSE_KNOWLEDGE) down -v
	@echo "Removing built images..."
	docker rmi $$(docker images | grep -E '(web-api|integration-api|endpoint-api|assistant-api|document-api|ui)' | awk '{print $$3}') 2>/dev/null || true
	@echo "✓ Cleanup complete"

# ============================================================================
# QUICK DEVELOPMENT COMMANDS
# ============================================================================

# Start core dependencies (db, redis) without APIs
deps:
	@echo "Starting core dependencies only..."
	$(COMPOSE) up -d postgres redis
	@echo "✓ Dependencies started"

# Start all dependencies including opensearch (for knowledge base)
deps-knowledge:
	@echo "Starting all dependencies including opensearch..."
	$(COMPOSE_KNOWLEDGE) up -d postgres redis opensearch
	@echo "✓ Dependencies started (with OpenSearch)"

# Start full stack with UI
full: build-all up-all

# Development mode - start with rebuild
dev: rebuild-all up-all logs-all



# ============================================================================
# RUN WITHOUT DOCKER TARGETS
# ============================================================================



run-document:
	@echo "Running document-api without Docker..."
	PYTHONPATH=api/document-api uvicorn app.main:app --host 0.0.0.0 --port 9010

run-assistant:
	@echo "Running assistant-api without Docker..."
	go run cmd/assistant/assistant.go

run-web:
	@echo "Running web-api without Docker..."
	go run cmd/web/web.go

run-endpoint:
	@echo "Running endpoint-api without Docker..."
	go run cmd/endpoint/endpoint.go

run-integration:
	@echo "Running integration-api without Docker..."
	go run cmd/integration/integration.go
	
run-ui:
	@echo "Running UI with yarn start in ui folder..."
	cd ui && yarn start
# ============================================================================
# INTEGRATION TESTS
# ============================================================================

test-tts-integration:
	@echo "Running TTS integration tests (requires testdata/integration_config.yaml)..."
	go test -tags integration -v -run TestTTS -timeout 120s ./api/assistant-api/internal/transformer/...

test-stt-integration:
	@echo "Running STT integration tests (requires testdata/integration_config.yaml)..."
	go test -tags integration -v -run TestSTT -timeout 120s ./api/assistant-api/internal/transformer/...

test-transformer-integration:
	@echo "Running all transformer integration tests..."
	go test -tags integration -v -timeout 180s ./api/assistant-api/internal/transformer/...

# Validate plan, implementation, verification, and independent review gates.
validate-development-process:
	@bash bin/validate-development-process

validate-agent-tooling:
	@bash bin/validate-agent-tooling

validate-rfc-layout:
	@python3 bin/validate-rfc-layout

validate-development-toolkit: validate-development-process validate-agent-tooling validate-rfc-layout

agent-finalize:
	@test -n "$(CHANGED_FILES)" || (echo "Usage: make agent-finalize CHANGED_FILES='path/a.go,path/a_test.go' [AGENT_ROOT=.codex]" && exit 2)
	@AGENT_ROOT="$(or $(AGENT_ROOT),.codex)" CHANGED_FILES="$(CHANGED_FILES)" bash bin/agent-finalize

sign-approved-plan:
	@test -n "$(RUN_ID)" -a -n "$(PLAN)" || (echo "Usage: DEVELOPMENT_GATE_KEY=... make sign-approved-plan RUN_ID='<orca-run-id>' PLAN='rfcs/NNNN-name/jsons/plan.json'" && exit 2)
	@bash bin/sign-approved-plan "$(RUN_ID)" "$(PLAN)"

orca-development-run:
	@test -n "$(OBJECTIVE)" -a -n "$(RFC)" || (echo "Usage: make orca-development-run OBJECTIVE='task objective' RFC='rfcs/NNNN-name.md' [AGENT=codex]" && exit 2)
	@bash bin/orca-development-run "$(OBJECTIVE)" "$(or $(AGENT),codex)" "$(RFC)"

orca-rfc-release:
	@test -n "$(RFC)" || (echo "Usage: make orca-rfc-release RFC='rfcs/NNNN-name.md'" && exit 2)
	@bash bin/orca-rfc-release "$(RFC)"

orca-confirm-rfc-create:
	@test -n "$(RUN_ID)" -a -n "$(CHALLENGE_TASK_ID)" -a -n "$(CHALLENGE_RECEIPT)" -a -n "$(RFC)" || (echo "Usage: make orca-confirm-rfc-create RUN_ID=... CHALLENGE_TASK_ID=... CHALLENGE_RECEIPT=rfcs/NNNN-name/jsons/challenge.json RFC=rfcs/NNNN-name.md" && exit 2)
	@python3 bin/orca-confirm-rfc create --run "$(RUN_ID)" --challenge-task "$(CHALLENGE_TASK_ID)" --challenge-receipt "$(CHALLENGE_RECEIPT)" --rfc "$(RFC)"

orca-confirm-rfc-collect:
	@test -n "$(RUN_ID)" -a -n "$(TASK_ID)" -a -n "$(CHALLENGE_TASK_ID)" -a -n "$(CHALLENGE_RECEIPT)" -a -n "$(GATE_ID)" -a -n "$(RFC)" || (echo "Usage: make orca-confirm-rfc-collect RUN_ID=... TASK_ID=... CHALLENGE_TASK_ID=... CHALLENGE_RECEIPT=rfcs/NNNN-name/jsons/challenge.json GATE_ID=... RFC=rfcs/NNNN-name.md [RECEIPT=rfcs/NNNN-name/jsons/confirmation.json]" && exit 2)
	@python3 bin/orca-confirm-rfc collect --run "$(RUN_ID)" --task "$(TASK_ID)" --challenge-task "$(CHALLENGE_TASK_ID)" --challenge-receipt "$(CHALLENGE_RECEIPT)" --gate "$(GATE_ID)" --rfc "$(RFC)" $(if $(RECEIPT),--output "$(RECEIPT)",)

orca-panel:
	@python3 bin/orca-development-panel --input "$(or $(PANEL_INPUT),.codex/orchestrator/examples/lifecycle-input.json)" --output "$(or $(PANEL_OUTPUT),.cache/orca-development-panel.html)"

orca-panel-open:
	@python3 bin/orca-development-panel --input "$(or $(PANEL_INPUT),.codex/orchestrator/examples/lifecycle-input.json)" --output "$(or $(PANEL_OUTPUT),.cache/orca-development-panel.html)" --open

# Add appropriate aliases for clarity
run-{service-name}:
	@echo "Please specify a valid service name: document-api, assistant, web, endpoint, or integration."
