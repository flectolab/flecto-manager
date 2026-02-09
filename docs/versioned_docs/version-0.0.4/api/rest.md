---
sidebar_position: 2
---

# REST API

Flecto Manager provides a REST API for agents to sync configurations.

## Base URL

```
https://your-manager.example.com/api
```

## Endpoints

### Get Project Version

Check if the project configuration has changed.

```http
GET /api/namespace/:namespace/project/:project/version
Authorization: Bearer <token>
```

**Response:**

```json
"1"
```

The version string changes whenever redirects or pages are published. Agents can use this to determine if they need to fetch updated configurations.

---

### Get Redirects

Fetch all published redirects for a project.

```http
GET /api/namespace/:namespace/project/:project/redirects
Authorization: Bearer <token>
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 500 | Maximum number of items to return |
| `offset` | int | 0 | Number of items to skip |

**Response:**

```json
{
  "items": [
    {
      "type": "BASIC",
      "source": "/old-page",
      "target": "/new-page",
      "status": "MOVED_PERMANENT"
    },
    {
      "type": "BASIC_HOST",
      "source": "example.com/shop",
      "target": "https://shop.example.com",
      "status": "FOUND"
    },
    {
      "type": "REGEX",
      "source": "^/blog/([0-9]+)/(.*)$",
      "target": "/articles/$1/$2",
      "status": "MOVED_PERMANENT"
    }
  ],
  "total": 3,
  "limit": 500,
  "offset": 0
}
```

---

### Get Pages

Fetch all published pages for a project.

```http
GET /api/namespace/:namespace/project/:project/pages
Authorization: Bearer <token>
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 500 | Maximum number of items to return |
| `offset` | int | 0 | Number of items to skip |

**Response:**

```json
{
  "items": [
    {
      "type": "BASIC",
      "path": "/robots.txt",
      "content": "User-agent: *\nAllow: /",
      "contentType": "TEXT_PLAIN"
    },
    {
      "type": "BASIC_HOST",
      "path": "shop.example.com/robots.txt",
      "content": "User-agent: *\nDisallow: /checkout/",
      "contentType": "TEXT_PLAIN"
    }
  ],
  "total": 2,
  "limit": 500,
  "offset": 0
}
```

---

### Register/Update Agent

Register an agent or update its information.

```http
POST /api/namespace/:namespace/project/:project/agents
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "traefik-eu-1",
  "type": "traefik",
  "version": 1,
  "status": "success",
  "load_duration": 150000000,
  "error": ""
}
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Agent name (alphanumeric, underscores, hyphens only) |
| `type` | string | Yes | Agent type: `default` or `traefik` |
| `version` | int | Yes | Configuration version loaded by the agent |
| `status` | string | No | Sync status: `success` or `error` |
| `load_duration` | int | No | Time to load configuration in nanoseconds |
| `error` | string | No | Error message if status is `error` |

**Response:**

```http
HTTP/1.1 200 OK
```

---

### Agent Heartbeat

Update the agent's last seen timestamp.

```http
PATCH /api/namespace/:namespace/project/:project/agents/:name/hit
Authorization: Bearer <token>
```

**Response:**

```http
HTTP/1.1 200 OK
```

---

### Health Check

Check if the Manager is running.

```http
GET /health/ping
```

**Response:**

```http
HTTP/1.1 204 No Content
```

## Data Types Reference

### Redirect Types

| Type | Description |
|------|-------------|
| `BASIC` | Exact path matching |
| `BASIC_HOST` | Exact path matching with host (source includes host) |
| `REGEX` | Regular expression matching on path |
| `REGEX_HOST` | Regular expression matching with host |

### Redirect Status

| Status | HTTP Code |
|--------|-----------|
| `MOVED_PERMANENT` | 301 |
| `FOUND` | 302 |
| `TEMPORARY_REDIRECT` | 307 |
| `PERMANENT_REDIRECT` | 308 |

### Page Types

| Type | Description |
|------|-------------|
| `BASIC` | Exact path matching |
| `BASIC_HOST` | Exact path matching with host (path includes host) |

### Page Content Types

| Content Type | MIME Type |
|--------------|-----------|
| `TEXT_PLAIN` | `text/plain` |
| `XML` | `application/xml` |

## Pagination

List endpoints support pagination:

```http
GET /api/namespace/prod/project/website/redirects?limit=100&offset=200
```

Check the `total` field to determine if more items exist:

```json
{
  "items": [...],
  "total": 350,
  "limit": 100,
  "offset": 200
}
```

If `offset + items.length < total`, more items are available.
