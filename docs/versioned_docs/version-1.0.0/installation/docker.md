---
sidebar_position: 1
---

# Docker Installation

The recommended way to run Flecto Manager in production.

## Quick Start

```bash
docker run -d \
  --name flecto-manager \
  -p 8080:8080 \
  -e FLECTO_MANAGER_CFG='
http:
  listen: "0.0.0.0:8080"
db:
  type: mysql
  config:
    dsn: "user:password@tcp(mysql:3306)/flecto?parseTime=true"
auth:
  jwt:
    secret: "your-secret-key-minimum-32-characters"
' \
  ghcr.io/flectolab/flecto-manager:1.0.0

# Initialize database (first time only)
docker exec flecto-manager flecto-manager db apply
docker exec flecto-manager flecto-manager db init

# (Optional) Add demo data for testing
docker exec flecto-manager flecto-manager db demo
```

## Docker Compose

For production deployments, use Docker Compose:

```yaml title="docker-compose.yml"
services:
  flecto-manager:
    image: ghcr.io/flectolab/flecto-manager:1.0.0
    container_name: flecto-manager
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/etc/flecto/manager.yaml:ro
    environment:
      - LOG_LEVEL=INFO
    depends_on:
      mysql:
        condition: service_healthy

  mysql:
    image: mysql:8.0
    container_name: flecto-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: flecto
      MYSQL_USER: flecto
      MYSQL_PASSWORD: flectopassword
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 10

volumes:
  mysql-data:
```

```yaml title="config.yaml"
http:
  listen: "0.0.0.0:8080"

db:
  type: mysql
  config:
    dsn: "flecto:flectopassword@tcp(mysql:3306)/flecto?parseTime=true"

auth:
  jwt:
    secret: "change-this-to-a-secure-32-char-secret"
    issuer: "flecto-manager"
    access_token_ttl: 15m
    refresh_token_ttl: 24h
```

Start the service:

```bash
docker compose up -d

# Wait for MySQL to be ready, then initialize the database
docker compose exec flecto-manager flecto-manager db apply
docker compose exec flecto-manager flecto-manager db init

# (Optional) Add demo data for testing
docker compose exec flecto-manager flecto-manager db demo
```

## Upgrading

When upgrading to a new version:

```bash
# Pull the new image
docker compose pull

# Restart with the new image
docker compose up -d

# Apply database migrations
docker compose exec flecto-manager flecto-manager db apply
```

## Platforms

Docker images are available for:
- `linux/amd64`
- `linux/arm64`
