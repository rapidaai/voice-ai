.PHONY: help up down build logs clean restart ps

help:
	@echo "Web-API Service Management"
	@echo "=========================="
	@echo "make up              - Start web-api with PostgreSQL and Redis"
	@echo "make down            - Stop all services"
	@echo "make build           - Build web-api image"
	@echo "make rebuild         - Rebuild web-api image (no cache)"
	@echo "make logs            - View all service logs"
	@echo "make logs-api        - View web-api logs only"
	@echo "make logs-db         - View PostgreSQL logs"
	@echo "make logs-redis      - View Redis logs"
	@echo "make clean           - Stop and remove all containers, volumes"
	@echo "make restart         - Restart all services"
	@echo "make ps              - Show running containers"
	@echo "make shell           - Open web-api container shell"
	@echo "make db-web        - Open PostgreSQL shell"

up:
	cd docker && docker-compose up -d

down:
	cd docker && docker-compose down

build:
	cd docker && docker-compose build web-api

rebuild:
	cd docker && docker-compose build --no-cache web-api

logs:
	cd docker && docker-compose logs -f

logs-api:
	cd docker && docker-compose logs -f web-api

logs-db:
	cd docker && docker-compose logs -f postgres

logs-redis:
	cd docker && docker-compose logs -f redis

clean:
	cd docker && docker-compose down -v

restart:
	cd docker && docker-compose restart

ps:
	cd docker && docker-compose ps

shell-web:
	cd docker && docker-compose exec web-api sh

db-web:
	cd docker && docker-compose exec postgres psql -U rapida_user -d web_api

# Development helpers
dev-up:
	@echo "Starting development environment..."
	make up
	@echo "Services are running!"
	@echo "Web-API: http://localhost:9001"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"

dev-logs:
	@echo "Following logs... (Ctrl+C to stop)"
	make logs