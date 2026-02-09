# Flecto Manager

<p align="center">
  <img src="webui/src/assets/flecto-full-light.svg" alt="Flecto Manager Logo" width="400">
</p>

<p align="center">
  <strong>Dynamic HTTP redirect and traffic routing management for modern infrastructure.</strong>
</p>

<p align="center">
  <a href="https://github.com/flectolab/flecto-manager/actions/workflows/app.yml"><img src="https://github.com/flectolab/flecto-manager/actions/workflows/app.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/flectolab/flecto-manager/releases"><img src="https://img.shields.io/github/v/release/flectolab/flecto-manager" alt="Release"></a>
  <a href="https://github.com/flectolab/flecto-manager/pkgs/container/flecto-manager"><img src="https://img.shields.io/badge/ghcr.io-flecto--manager-blue" alt="Docker"></a>
</p>

---

## What is Flecto Manager?

Flecto Manager is a centralized management platform for HTTP redirections and static pages. It provides a web interface to configure routing rules that are then consumed by **agents** deployed in your infrastructure.

### Features

**Redirections**
- Manage HTTP redirects (301, 302, 307, 308)
- Support for path patterns and query string handling
- Bulk import/export via CSV
- Draft mode for reviewing changes before publishing

**Pages**
- Serve static text content (`text/plain`) for files like `robots.txt`
- Serve XML content (`text/xml`) for files like `sitemap.xml`
- Version control with draft/publish workflow

### Architecture

```
┌─────────────────┐         ┌─────────────────┐
│  Flecto Manager │◄────────│   Web UI        │
│     (API)       │         │  (React)        │
└────────┬────────┘         └─────────────────┘
         │
         │ HTTP API (pull config)
         │
         ▼
┌─────────────────┐         ┌─────────────────┐
│     Agent       │────────►│  Reverse Proxy  │
│   (Traefik,     │         │  (serves users) │
│    custom...)   │         │                 │
└─────────────────┘         └─────────────────┘
```

**How it works:**
1. Configure redirects and pages in Flecto Manager (via Web UI or GraphQL API)
2. Deploy an **agent** in your infrastructure (e.g., Traefik plugin, custom agent)
3. The agent periodically fetches the configuration from Flecto Manager
4. The agent applies redirects and serves pages to end users

> **Note:** Flecto Manager is the control plane only. You need an agent to actually process redirects and serve pages to users.

---

## Prerequisites

- **Go** 1.24+
- **Node.js** 22+ (see `webui/.nvmrc`)
- **npm** 10+

## Project Structure

```
├── common/          # Shared Go package (can be imported separately)
├── webui/           # React frontend (TypeScript + Vite)
├── cli/             # CLI commands (Cobra)
├── http/            # HTTP server (Echo + GraphQL)
├── service/         # Business logic
├── repository/      # Data access layer
├── model/           # Database models (GORM)
└── graph/           # GraphQL schema & resolvers
```

## Quick Start

### Using Docker

```bash
docker pull ghcr.io/flectolab/flecto-manager:latest

docker run -p 8080:8080 -e FLECTO_MANAGER_CFG='
http:
  listen: "0.0.0.0:8080"
db:
  type: mysql
  config:
    dsn: "user:password@tcp(localhost:3306)/flecto?parseTime=true"
auth:
  jwt:
    secret: "your-secret-key-at-least-32-chars!"
' ghcr.io/flectolab/flecto-manager:latest
```

### From Source

```bash
# Clone repository
git clone https://github.com/flectolab/flecto-manager.git
cd flecto-manager

# Build frontend
cd webui
npm ci
npm run codegen
npm run build
cd ..

# Build and run
go mod download
go tool gqlgen generate
./bin/mock.sh
go build -o flecto-manager .
./flecto-manager start -c config.yaml
```

---

## Development

### Frontend (webui)

```bash
cd webui

# Install dependencies
npm ci

# Generate GraphQL types
npm run codegen

# Development server (hot reload)
npm run dev

# Production build
npm run build

# Lint
npm run lint
```

The frontend runs on `http://localhost:5173` by default.

### Backend (Go)

```bash
# Download dependencies
go mod download

# Generate GraphQL resolvers
go tool gqlgen generate

# Generate mocks for testing
./bin/mock.sh

# Run tests
go test -v ./...

# Build
go build -v .

# Run
./flecto-manager start -c config.yaml
```

### Common Package

The `common/` package is a standalone Go module that can be imported in other projects.

```bash
cd common

# Download dependencies
go mod download

# Run tests
go test -v -race ./...
```

**Import in other projects:**

```bash
go get github.com/flectolab/flecto-manager/common@latest
```

---

## Configuration

Create a `config.yaml` file:

```yaml
http:
  listen: "127.0.0.1:8080"

db:
  type: mysql
  config:
    dsn: "user:password@tcp(localhost:3306)/flecto?parseTime=true"

auth:
  jwt:
    secret: "your-secret-key-at-least-32-characters"
    issuer: "flecto-manager"
    access_token_ttl: 15m
    refresh_token_ttl: 24h
    header_name: "Authorization"

page:
  size_limit: 1048576        # 1MB
  total_size_limit: 104857600 # 100MB

agent:
  offline_threshold: 6h
```

### Environment Variables

When using Docker, you can inject the config via environment variable:

| Variable | Description |
|----------|-------------|
| `FLECTO_MANAGER_CFG` | Full YAML configuration content |
| `FLECTO_MANAGER_CONFIG_PATH` | Config file path (default: `/etc/flecto/manager.yaml`) |
| `LOG_LEVEL` | Log level: DEBUG, INFO, WARN, ERROR (default: `INFO`) |

---

## API

- **GraphQL Playground**: `http://localhost:8080/graphql`
- **REST API**: `http://localhost:8080/api/...`

Default credentials: `admin` / `admin`

---

## License

MIT
