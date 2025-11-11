.PHONY: help up down build rebuild logs clean restart ps shell db-shell \
        up-all up-web up-integration up-endpoint up-db up-redis up-opensearch \
        down-all down-web down-integration down-endpoint down-db down-redis down-opensearch \
        build-all build-web build-integration build-endpoint \
        rebuild-all rebuild-web rebuild-integration rebuild-endpoint \
        logs-all logs-web logs-integration logs-endpoint logs-db logs-redis logs-opensearch \
        restart-all restart-web restart-integration restart-endpoint \
        ps-all shell-web shell-integration shell-endpoint db-shell

COMPOSE := docker compose -f docker-compose.yml

help:
	@echo ""
	@echo "╔════════════════════════════════════════════════════════════════╗"
	@echo "║          Docker Compose Service Management                    ║"
	@echo "╚════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "STARTUP COMMANDS:"
	@echo "  make up-all              - Start all services"
	@echo "  make up-web              - Start web-api only"
	@echo "  make up-integration      - Start integration-api only"
	@echo "  make up-endpoint         - Start endpoint-api only"
	@echo "  make up-db               - Start PostgreSQL only"
	@echo "  make up-redis            - Start Redis only"
	@echo "  make up-opensearch       - Start OpenSearch only"
	@echo "  make up-nginx       - Start nginx only"
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
	@echo "  make build-all           - Build all services"
	@echo "  make build-web           - Build web-api image"
	@echo "  make build-integration   - Build integration-api image"
	@echo "  make build-endpoint      - Build endpoint-api image"
	@echo "  make rebuild-all         - Rebuild all (no cache)"
	@echo "  make rebuild-web         - Rebuild web-api (no cache)"
	@echo "  make rebuild-integration - Rebuild integration-api (no cache)"
	@echo "  make rebuild-endpoint    - Rebuild endpoint-api (no cache)"
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
	@echo "  make db-shell            - Open PostgreSQL shell"
	@echo ""
	@echo "MAINTENANCE:"
	@echo "  make clean               - Stop and remove containers, volumes, images"
	@echo "  make clean-volumes       - Remove only volumes"
	@echo "  make status              - Show container status and ports"
	@echo ""

# ============================================================================
# STARTUP TARGETS - Individual Services
# ============================================================================

up-all:
	@echo "Starting all services..."
	$(COMPOSE) up -d
	@echo "✓ All services started"
	@$(MAKE) status

up-ui:
	@echo "Starting ui..."
	$(COMPOSE) up -d ui
	@echo "✓ ui started on port 3000"

up-document:
	@echo "Starting document-api..."
	$(COMPOSE) up -d document-api
	@echo "✓ web-api started on port 9010"

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
	@echo "✓ assistant-api started on port 9005"

up-db:
	@echo "Starting PostgreSQL..."
	$(COMPOSE) up -d postgres
	@echo "✓ PostgreSQL started on port 5432"

up-nginx:
	@echo "Starting nginx..."
	$(COMPOSE) up -d nginx
	@echo "✓ nginx started on port 6379"

up-redis:
	@echo "Starting Redis..."
	$(COMPOSE) up -d redis
	@echo "✓ Redis started on port 6379"

up-opensearch:
	@echo "Starting OpenSearch..."
	$(COMPOSE) up -d opensearch
	@echo "✓ OpenSearch started on port 9200"

# Legacy aliases
up: up-all

# ============================================================================
# SHUTDOWN TARGETS - Individual Services
# ============================================================================

down-all:
	@echo "Stopping all services..."
	$(COMPOSE) down
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
	$(COMPOSE) stop document-api
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
	$(COMPOSE) stop opensearch
	@echo "✓ OpenSearch stopped"

# Legacy alias
down: down-all

# ============================================================================
# BUILD TARGETS
# ============================================================================

build-all:
	@echo "Building all services..."
	$(COMPOSE) build ui web-api integration-api endpoint-api
	@echo "✓ All services built"

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
	$(COMPOSE) build document-api
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
	@echo "Rebuilding all services (no cache)..."
	$(COMPOSE) build --no-cache ui web-api integration-api endpoint-api
	@echo "✓ All services rebuilt"

rebuild-web:
	@echo "Rebuilding web-api (no cache)..."
	$(COMPOSE) build --no-cache web-api
	@echo "✓ web-api rebuilt"

rebuild-document:
	@echo "Rebuilding document-api (no cache)..."
	$(COMPOSE) build --no-cache document-api
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
	$(COMPOSE) logs -f

logs-ui:
	$(COMPOSE) logs -f ui

logs-web:
	$(COMPOSE) logs -f web-api


logs-document:
	$(COMPOSE) logs -f document-api


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
	$(COMPOSE) logs -f opensearch

# Legacy alias
logs: logs-all

# ============================================================================
# RESTART TARGETS
# ============================================================================

restart-all:
	@echo "Restarting all services..."
	$(COMPOSE) restart
	@echo "✓ All services restarted"

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
	$(COMPOSE) restart document-api
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
	@echo "  Web-API:          http://localhost:9001"
	@echo "  Integration-API:  http://localhost:9004"
	@echo "  Endpoint-API:     http://localhost:9005"
	@echo "  PostgreSQL:       localhost:5432"
	@echo "  Redis:            localhost:6379"
	@echo "  OpenSearch:       https://localhost:9200"
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
	$(COMPOSE) exec document-api sh

shell-web:
	$(COMPOSE) exec web-api sh

shell-integration:
	$(COMPOSE) exec integration-api sh

shell-endpoint:
	$(COMPOSE) exec endpoint-api sh

db-shell:
	$(COMPOSE) exec postgres psql -U rapida_user -d web_db

# Legacy alias
shell: shell-web

# ============================================================================
# MAINTENANCE TARGETS
# ============================================================================

clean-volumes:
	@echo "Removing volumes..."
	$(COMPOSE) down -v
	@echo "✓ Volumes removed"

clean:
	@echo "Cleaning up Docker resources..."
	$(COMPOSE) down -v
	@echo "Removing built images..."
	docker rmi $$(docker images | grep -E '(web-api|integration-api|endpoint-api)' | awk '{print $$3}') 2>/dev/null || true
	@echo "✓ Cleanup complete"

# ============================================================================
# QUICK DEVELOPMENT COMMANDS
# ============================================================================

# Start all dependencies (db, redis, opensearch) without APIs
deps:
	@echo "Starting dependencies only..."
	$(COMPOSE) up -d postgres redis opensearch
	@echo "✓ Dependencies started"

# Start full stack with UI
full: build-all up-all

# Development mode - start with rebuild
dev: rebuild-all up-all logs-all