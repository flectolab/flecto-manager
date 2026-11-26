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
| `cursor` | string | - | Opaque cursor from a previous response, see [Pagination](#pagination). Replaces `offset` when set |

**Response:**

```json
{
  "Items": [
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
  "Total": 3,
  "Limit": 500,
  "Offset": 0
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
| `cursor` | string | - | Opaque cursor from a previous response, see [Pagination](#pagination). Replaces `offset` when set |

**Response:**

```json
{
  "Items": [
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
  "Total": 2,
  "Limit": 500,
  "Offset": 0
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

List endpoints support two ways of walking a listing. Both return the same fields,
so a client can switch from one to the other without changing how it reads the
response.

### By offset

```http
GET /api/namespace/prod/project/website/redirects?limit=100&offset=200
```

```json
{
  "Items": [...],
  "Total": 350,
  "Limit": 100,
  "Offset": 200,
  "Next": "eyJpZCI6MzAwLCJ0b3RhbCI6MzUwLCJkZWxpdmVyZWQiOjMwMH0"
}
```

If `Offset + Items.length < Total`, more items are available.

### By cursor (recommended for large projects)

`Next` is an opaque cursor pointing just past the last item of the response. Pass it
back as the `cursor` parameter to get the following page:

```http
GET /api/namespace/prod/project/website/redirects?limit=100&cursor=eyJpZCI6MzAwLCJ0b3RhbCI6MzUwLCJkZWxpdmVyZWQiOjMwMH0
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `cursor` | string | Opaque value returned as `Next` by the previous response. Replaces `offset`, which is ignored when both are sent. |

An empty or absent `Next` means the listing is over. A malformed cursor is rejected
with `400 Bad Request`.

Offset pagination has to skip the rows it does not return, so page 1000 costs more
than page 1. A cursor walks from a position instead, which keeps every page the same
cost, and it carries the total measured on the first page so the count query is not
repeated. On a project with hundreds of thousands of redirects this is the difference
between a sync that slows down as it goes and one that does not.

:::note
Treat the cursor as opaque: its content is a server implementation detail and may
change. Do not build one by hand, only echo back the value you received.
:::
