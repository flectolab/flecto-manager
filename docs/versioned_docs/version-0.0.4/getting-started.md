---
sidebar_position: 2
---

# Getting Started

This guide will help you install and run Flecto Manager.

## Prerequisites

- Docker (recommended) or Go 1.24+
- MySQL 8.0+ database
- A reverse proxy (nginx, Caddy, etc.) for production

## Quick Start with Docker

The fastest way to get started is using Docker:

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
```

## Database Initialization

After starting the container for the first time, initialize the database:

```bash
# Apply database migrations
docker exec flecto-manager flecto-manager db apply

# Initialize default data (admin user, etc.)
docker exec flecto-manager flecto-manager db init

# (Optional) Add demo data for testing
docker exec flecto-manager flecto-manager db demo
```

The Manager will be available at `http://localhost:8080`.

**Default credentials:**
- Username: `admin`
- Password: `admin`

:::warning
Change the default password immediately after first login!
:::

## Configuration

Create a configuration file to customize the Manager:

```yaml title="config.yaml"
http:
  listen: "0.0.0.0:8080"

db:
  type: mysql
  config:
    dsn: "user:password@tcp(localhost:3306)/flecto?parseTime=true"

auth:
  jwt:
    secret: "your-secret-key-minimum-32-characters"
    issuer: "flecto-manager"
    access_token_ttl: 15m
    refresh_token_ttl: 24h
```

Run with your configuration:

```bash
docker run -d \
  --name flecto-manager \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/flecto/manager.yaml \
  ghcr.io/flectolab/flecto-manager:1.0.0
```

## First Steps

1. **Login** with the default credentials
2. **Create a Namespace** to organize your projects (e.g., `production`, `staging`)
3. **Create a Project** within the namespace (e.g., `example-com`)
4. **Add Redirections** or **Static Pages** to your project
5. **Deploy an Agent** to serve the configurations

## Next Steps

- [Docker Installation](./installation/docker) - Detailed Docker setup
- [Binary Installation](./installation/binary) - Install from binary
- [Configuration Reference](./configuration) - Full configuration options
