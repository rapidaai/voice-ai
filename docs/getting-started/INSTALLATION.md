# Installation

Get Rapida running locally in about 10 minutes.

---

## 1. Clone the Repository

```bash
git clone https://github.com/rapidaai/voice-ai.git
cd voice-ai
```

---

## 2. Create Data Directory

Rapida stores persistent data (database, files, cache) in a local directory.

```bash
mkdir -p ${HOME}/rapida-data/
```

**Linux only:** Grant Docker group access:

```bash
sudo setfacl -m g:docker:rwx ${HOME}/rapida-data/
```

**macOS:** No additional permissions needed. Docker Desktop handles this.

---

## 3. Build All Services

This builds Docker images for all Rapida services. Takes 5-10 minutes on first run.

```bash
make build-all
```

You'll see output for each service: `ui`, `web-api`, `assistant-api`, `integration-api`, `endpoint-api`, `document-api`.

---

## 4. Start the Platform

```bash
make up-all
```

This starts all services in the background. Wait about 30 seconds for everything to initialize.

Check status:

```bash
make status
```

You should see all containers running:

```
Running Containers:
===================
NAME              STATUS
postgres          Up (healthy)
redis             Up (healthy)
opensearch        Up (healthy)
web-api           Up (healthy)
assistant-api     Up (healthy)
integration-api   Up (healthy)
endpoint-api      Up (healthy)
document-api      Up (healthy)
ui                Up
nginx             Up
```

---

## 5. Verify Installation

Open your browser:

| Service | URL |
|---------|-----|
| **UI** | [http://localhost:3000](http://localhost:3000) |
| Web API | [http://localhost:9001/readiness/](http://localhost:9001/readiness/) |
| Assistant API | [http://localhost:9007/readiness/](http://localhost:9007/readiness/) |

The UI should show a login/signup page. The API endpoints should return a success response.

---

## Troubleshooting

### OpenSearch won't start

OpenSearch needs memory. Check Docker's memory allocation:
- **Docker Desktop:** Settings → Resources → Memory (set to 8GB+)

Also check system limits (Linux):

```bash
sudo sysctl -w vm.max_map_count=262144
```

### Port conflicts

If you see "port already in use" errors, find what's using the port:

```bash
lsof -i :3000  # Replace with the conflicting port
```

Either stop that process or modify `docker-compose.yml` to use different ports.

### View logs

```bash
make logs-all          # All services
make logs-web          # Web API only
make logs-assistant    # Assistant API only
```

### Rebuild after code changes

```bash
make rebuild-all       # Rebuild without cache
make restart-all       # Restart all services
```

### Clean start

To wipe everything and start fresh:

```bash
make down-all          # Stop all services
make clean-volumes     # Remove volumes (deletes data!)
make build-all         # Rebuild
make up-all            # Start fresh
```

---

## Common Commands

| Command | Purpose |
|---------|---------|
| `make up-all` | Start all services |
| `make down-all` | Stop all services |
| `make status` | Show running containers |
| `make logs-all` | View all logs |
| `make restart-all` | Restart all services |
| `make help` | Show all available commands |

---

## Running Without Docker (Development)

For local development, you can run services directly:

```bash
# Start dependencies first
make up-db            # PostgreSQL
make up-redis         # Redis
make up-opensearch    # OpenSearch

# Run services individually
make run-web          # go run cmd/web/web.go
make run-assistant    # go run cmd/assistant/assistant.go
make run-ui           # cd ui && yarn start
```

This requires Go 1.21+, Node.js 18+, and Python 3.11+ installed locally.

---

## Next Step

Platform running? Continue to [First Voice Agent](./FIRST_VOICE_AGENT.md) to create your first assistant.

