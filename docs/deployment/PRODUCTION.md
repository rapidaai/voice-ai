# Production Deployment

Deploy Rapida for production use. This guide covers Docker Compose deployments suitable for small to medium workloads.

For high-scale deployments, contact sales@rapida.ai for enterprise guidance.

---

## Prerequisites

- Completed [Installation](../getting-started/INSTALLATION.md) locally
- A server with Docker and Docker Compose
- Domain name (optional but recommended)
- SSL certificates (for HTTPS)

---

## 1. Server Requirements

### Minimum

| Resource | Requirement |
|----------|-------------|
| CPU | 4 cores |
| RAM | 16 GB |
| Disk | 50 GB SSD |
| OS | Ubuntu 22.04 LTS (or similar) |

### Recommended

| Resource | Requirement |
|----------|-------------|
| CPU | 8+ cores |
| RAM | 32 GB |
| Disk | 100 GB SSD |
| Network | Low latency to your users |

Voice applications are latency-sensitive. Choose a region close to your users.

---

## 2. Environment Configuration

### Create Environment Files

Each service needs its own environment file. Create these in `/opt/rapida/env/` or a secure location:

```bash
mkdir -p /opt/rapida/env
```

#### Web API (.web.env)

```bash
# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=rapida_user
POSTGRES_PASSWORD=<strong-password>
POSTGRES_DB=web_db

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Security
JWT_SECRET=<generate-a-long-random-string>
ENCRYPTION_KEY=<32-byte-hex-string>

# Service URLs (internal Docker network)
ASSISTANT_API_URL=http://assistant-api:9007
INTEGRATION_API_URL=http://integration-api:9004
ENDPOINT_API_URL=http://endpoint-api:9005
```

#### Assistant API (.assistant.env)

```bash
# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=rapida_user
POSTGRES_PASSWORD=<same-password>
POSTGRES_DB=assistant_db

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# OpenSearch
OPENSEARCH_HOST=opensearch
OPENSEARCH_PORT=9200

# Security
JWT_SECRET=<same-jwt-secret>

# Storage
STORAGE_PATH=/app/rapida-data/assets
```

#### Integration API (.integration.env)

```bash
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=rapida_user
POSTGRES_PASSWORD=<same-password>
POSTGRES_DB=integration_db

REDIS_HOST=redis
REDIS_PORT=6379

JWT_SECRET=<same-jwt-secret>
```

#### Endpoint API (.endpoint.env)

```bash
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=rapida_user
POSTGRES_PASSWORD=<same-password>
POSTGRES_DB=endpoint_db

REDIS_HOST=redis
REDIS_PORT=6379

JWT_SECRET=<same-jwt-secret>
```

### Generate Secrets

```bash
# JWT Secret (64 characters)
openssl rand -hex 32

# Encryption Key (32 bytes)
openssl rand -hex 16
```

**Important:** Use the same `JWT_SECRET` across all services. Use different, strong passwords for production.

---

## 3. Docker Compose Production Overrides

Create a `docker-compose.prod.yml` that overrides development settings:

```yaml
services:
  postgres:
    restart: always
    environment:
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - /var/lib/rapida/postgres:/var/lib/postgresql/data

  redis:
    restart: always
    command: redis-server --appendonly yes
    volumes:
      - /var/lib/rapida/redis:/data

  opensearch:
    restart: always
    environment:
      - "OPENSEARCH_JAVA_OPTS=-Xms1g -Xmx1g"
    volumes:
      - /var/lib/rapida/opensearch:/usr/share/opensearch/data

  web-api:
    restart: always
    env_file:
      - /opt/rapida/env/.web.env
    deploy:
      resources:
        limits:
          memory: 1G

  assistant-api:
    restart: always
    env_file:
      - /opt/rapida/env/.assistant.env
    deploy:
      resources:
        limits:
          memory: 2G

  integration-api:
    restart: always
    env_file:
      - /opt/rapida/env/.integration.env
    deploy:
      resources:
        limits:
          memory: 512M

  endpoint-api:
    restart: always
    env_file:
      - /opt/rapida/env/.endpoint.env
    deploy:
      resources:
        limits:
          memory: 512M

  document-api:
    restart: always
    deploy:
      resources:
        limits:
          memory: 1G

  ui:
    restart: always

  nginx:
    restart: always
    ports:
      - "80:8080"
      - "443:443"
    volumes:
      - /opt/rapida/nginx/nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - /opt/rapida/ssl:/etc/nginx/ssl:ro
```

### Run with Production Overrides

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

---

## 4. Reverse Proxy & SSL

### NGINX Configuration with SSL

Create `/opt/rapida/nginx/nginx.conf`:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    client_max_body_size 10m;

    # UI
    location / {
        proxy_pass http://ui:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # gRPC-Web APIs
    location ~ ^/(talk_api|assistant_api|web_api|endpoint_api|integration_api) {
        proxy_pass http://web-api:9001;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # CORS headers for gRPC-Web
        add_header 'Access-Control-Allow-Origin' '*' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' '*' always;
    }

    # WebSocket for voice streaming
    location ~ ^/(talk_api.TalkService) {
        proxy_pass http://assistant-api:9007;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

### SSL Certificates

Use Let's Encrypt with certbot:

```bash
apt install certbot
certbot certonly --standalone -d your-domain.com
cp /etc/letsencrypt/live/your-domain.com/fullchain.pem /opt/rapida/ssl/
cp /etc/letsencrypt/live/your-domain.com/privkey.pem /opt/rapida/ssl/
```

---

## 5. Health Checks & Monitoring

### Health Check Endpoints

| Service | Endpoint |
|---------|----------|
| Web API | `GET /readiness/` |
| Assistant API | `GET /readiness/` |
| Integration API | `GET /readiness/` |
| Endpoint API | `GET /readiness/` |
| Document API | `GET /readiness/` |

### Basic Monitoring Script

```bash
#!/bin/bash
# health-check.sh

services=("web-api:9001" "assistant-api:9007" "integration-api:9004" "endpoint-api:9005")

for service in "${services[@]}"; do
    name=$(echo $service | cut -d: -f1)
    port=$(echo $service | cut -d: -f2)
    
    if curl -sf "http://localhost:$port/readiness/" > /dev/null; then
        echo "✓ $name is healthy"
    else
        echo "✗ $name is unhealthy"
    fi
done
```

### Recommended: External Monitoring

Consider using:
- **Uptime monitoring:** UptimeRobot, Pingdom, or similar
- **Log aggregation:** Datadog, Logstash, or CloudWatch
- **Metrics:** Prometheus + Grafana

---

## 6. Backup Strategy

### What to Back Up

| Data | Location | Frequency |
|------|----------|-----------|
| PostgreSQL | `/var/lib/rapida/postgres` | Daily |
| Uploaded files | `/var/lib/rapida/assets` | Daily |
| Redis | `/var/lib/rapida/redis` | Optional (cache) |
| OpenSearch | `/var/lib/rapida/opensearch` | Weekly |

### PostgreSQL Backup Script

```bash
#!/bin/bash
# backup-postgres.sh

BACKUP_DIR="/var/backups/rapida"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

docker exec postgres pg_dumpall -U rapida_user > "$BACKUP_DIR/postgres_$DATE.sql"
gzip "$BACKUP_DIR/postgres_$DATE.sql"

# Keep last 7 days
find $BACKUP_DIR -name "postgres_*.sql.gz" -mtime +7 -delete
```

---

## 7. Scaling Considerations

### Horizontal Scaling

For higher load, run multiple instances of stateless services:

```yaml
# docker-compose.prod.yml
services:
  assistant-api:
    deploy:
      replicas: 3
```

Use a load balancer (nginx, HAProxy, or cloud LB) in front.

### Database Scaling

- **PostgreSQL:** Consider managed services (RDS, Cloud SQL) for automatic backups and replication
- **Redis:** Use Redis Cluster or managed Redis for high availability
- **OpenSearch:** Configure as a cluster for resilience

---

## 8. Security Checklist

- [ ] Change all default passwords
- [ ] Use strong, unique secrets for JWT and encryption
- [ ] Enable SSL/TLS (HTTPS only)
- [ ] Restrict database ports to internal network only
- [ ] Set up firewall rules (only expose 80/443)
- [ ] Enable automatic security updates on the server
- [ ] Review access logs regularly
- [ ] Set up alerting for failed health checks

---

## 9. Deployment Commands

```bash
# Initial deployment
git clone https://github.com/rapidaai/voice-ai.git /opt/rapida/voice-ai
cd /opt/rapida/voice-ai
docker compose -f docker-compose.yml -f docker-compose.prod.yml build
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Update to latest
cd /opt/rapida/voice-ai
git pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml build
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# View logs
docker compose logs -f assistant-api

# Restart a service
docker compose restart assistant-api
```

---

## Need Help?

- **Issues:** [GitHub Issues](https://github.com/rapidaai/voice-ai/issues)
- **Enterprise support:** sales@rapida.ai
- **Security concerns:** contact@rapida.ai

