---
sidebar_position: 3
---

# Agents

Agents are components that sync configurations from the Manager and serve HTTP requests. Flecto provides two official agent implementations.

## Architecture

```
                    ┌─────────────┐
                    │   Manager   │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
        ┌─────────┐  ┌─────────┐  ┌─────────┐
        │ Traefik │  │ Traefik │  │ Custom  │
        │ + Plugin│  │ + Plugin│  │  Agent  │
        └─────────┘  └─────────┘  └─────────┘
```

## Benefits

- **Low latency** - Deploy agents close to users
- **High availability** - Agents work independently
- **Automatic sync** - Configurations update automatically
- **Lightweight** - Minimal resource usage

## Traefik Middleware

The recommended way to deploy Flecto agents is using the [Traefik middleware plugin](https://github.com/flectolab/flecto-traefik-middleware).

### Installation

Add the plugin to your Traefik static configuration:

```yaml title="traefik.yml"
experimental:
  plugins:
    flecto:
      moduleName: github.com/flectolab/flecto-traefik-middleware
      version: v1.0.0
```

### Basic Configuration (Single Project)

Configure the middleware for a single project:

```yaml title="dynamic/flecto.yml"
http:
  middlewares:
    flecto:
      plugin:
        flecto:
          manager_url: "https://flecto-manager.example.com"
          namespace_code: "production"
          project_code: "my-website"
          token_jwt: "your-api-token"
          interval_check: "5m"
          agent_name: "traefik-eu-1"
```

### Multi-Host Configuration

When different hosts need different Flecto projects:

```yaml title="dynamic/flecto.yml"
http:
  middlewares:
    flecto:
      plugin:
        flecto:
          manager_url: "https://flecto-manager.example.com"
          namespace_code: "production"
          token_jwt: "your-api-token"
          host_configs:
            - hosts:
                - "example.com"
                - "www.example.com"
              project_code: "example-com"
            - hosts:
                - "shop.example.com"
              project_code: "shop"
```

### Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `manager_url` | Yes | - | Flecto Manager API URL |
| `namespace_code` | Yes | - | Namespace identifier |
| `project_code` | Yes* | - | Project identifier (*not required if using `host_configs`) |
| `token_jwt` | Yes | - | API token for authentication |
| `interval_check` | No | `5m` | How often to sync configurations |
| `agent_name` | No | - | Agent name shown in Manager dashboard |
| `header_authorization_name` | No | `Authorization` | Header name for token |
| `debug` | No | `false` | Enable debug logging |

### How It Works

1. Middleware connects to Flecto Manager on startup
2. Polls periodically for configuration updates
3. Matches incoming requests against redirect rules and pages
4. Either redirects with appropriate HTTP status or passes to next handler

## Go Client Library

For building custom agents, use the [Go client library](https://github.com/flectolab/go-client).

### Installation

```bash
go get github.com/flectolab/go-client
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    flecto "github.com/flectolab/go-client"
)

func main() {
    client := flecto.NewClient(&flecto.Config{
        ManagerURL:    "https://flecto-manager.example.com",
        NamespaceCode: "production",
        ProjectCode:   "my-website",
        AgentType:     "custom",
        TokenJWT:      "your-api-token",
        AgentName:     "custom-agent-1",
        CheckInterval: "5m",
    })

    // Initialize and load configuration
    if err := client.Init(); err != nil {
        log.Fatal(err)
    }

    // Start background refresh
    ctx := context.Background()
    client.Start(ctx)

    // Match redirects
    redirect := client.RedirectMatch("example.com", "/old-page")
    if redirect != nil {
        // Handle redirect
        log.Printf("Redirect to: %s (status: %d)", redirect.Target, redirect.HTTPCode())
    }

    // Match pages
    page := client.PageMatch("example.com", "/robots.txt")
    if page != nil {
        // Serve page content
        log.Printf("Serve page: %s (content-type: %s)", page.Path, page.HTTPContentType())
    }
}
```

### Features

- Fetches and caches redirect rules and pages
- Automatic periodic refresh based on project version changes
- Only fetches new data when changes are detected
- Thread-safe for concurrent access

## Monitoring

The Manager dashboard shows agent status:

- **Status** - Online/Offline
- **Last Sync** - When the agent last synced
- **Version** - Agent version
- **Name** - Agent identifier

Agents are marked offline after the configured threshold (default: 6 hours).

## Failover

If an agent cannot reach the Manager:

1. Agent continues serving from local cache
2. Agent retries connection with exponential backoff
3. Once connected, agent syncs latest configuration

This ensures redirects and pages continue working even if the Manager is temporarily unavailable.
