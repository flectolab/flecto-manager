# Flecto Manager Load Tests

Load testing suite for Flecto Manager API using [k6](https://k6.io/).

## Prerequisites

- [k6](https://k6.io/docs/getting-started/installation/) installed
- Flecto Manager server running

## Setup

1. Copy the config template:

```bash
cp config/config.dist.json config/config.json
```

2. Edit `config/config.json` with your settings:
   - `baseUrl`: Server URL
   - `auth`: Authentication credentials
   - `projects`: List of namespace/project pairs to test
   - `agents`: List of agent configurations for agent tests
   - `scenarios`: Configure each test scenario

## Configuration

### Projects

Define the namespace/project pairs to test against. Tests will randomly select from this list:

```json
{
  "projects": [
    { "namespace": "production", "project": "website" },
    { "namespace": "staging", "project": "api" }
  ]
}
```

### Agents

Define agent configurations for agent tests:

```json
{
  "agents": [
    { "name": "agent-1", "hostname": "server-1.example.com" },
    { "name": "agent-2", "hostname": "server-2.example.com" }
  ]
}
```

### Scenarios

Each scenario can be configured with:

| Field | Description |
|-------|-------------|
| `enabled` | Enable/disable the scenario |
| `executor` | k6 executor type (e.g., `constant-arrival-rate`) |
| `rate` | Request rate |
| `timeUnit` | Time unit for rate (e.g., `1s`) |
| `duration` | Test duration (e.g., `1m`) |
| `preAllocatedVUs` | Pre-allocated virtual users |
| `maxVUs` | Maximum virtual users |
| `thresholds.responseTime` | P95 response time threshold (ms) |

For `pages` and `redirects` scenarios:

| Field | Description |
|-------|-------------|
| `limit` | Page size for pagination (default: 50) |

## Available Scenarios

| Scenario | Description | Endpoint |
|----------|-------------|----------|
| `version` | Get project version | `GET /api/namespace/:ns/project/:proj/version` |
| `pages` | Fetch all pages (paginated) | `GET /api/namespace/:ns/project/:proj/pages` |
| `redirects` | Fetch all redirects (paginated) | `GET /api/namespace/:ns/project/:proj/redirects` |
| `agentHit` | Update agent last hit | `PATCH /api/namespace/:ns/project/:proj/agents/:name/hit` |
| `agentPost` | Create/update agent | `POST /api/namespace/:ns/project/:proj/agents` |

## Running Tests

### Run all enabled scenarios

```bash
k6 run load-api-agent.js
```

### Run a single scenario independently

Each test can be run standalone:

```bash
# Run version test only
k6 run tests/version.js

# Run pages test only
k6 run tests/pages.js

# Run redirects test only
k6 run tests/redirects.js

# Run agent hit test only
k6 run tests/agent-hit.js

# Run agent post test only
k6 run tests/agent-post.js
```

### Override base URL

```bash
k6 run -e BASE_URL=https://api.example.com load-api-agent.js
```

### Override authentication

```bash
k6 run -e AUTH_USERNAME=user -e AUTH_PASSWORD=pass load-api-agent.js
```

### Output to JSON

```bash
k6 run --out json=results.json load-api-agent.js
```

### Output to InfluxDB

```bash
k6 run --out influxdb=http://localhost:8086/k6 load-api-agent.js
```

## Metrics

### Custom Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `version_duration` | Trend | Response time for version endpoint |
| `version_requests` | Counter | Total version requests |
| `pages_duration` | Trend | Response time per paginated request |
| `pages_total_duration` | Trend | Total time to fetch all pages (full pagination) |
| `pages_requests` | Counter | Total pages requests |
| `pages_items_total_fetched` | Counter | Total page items fetched |
| `redirects_duration` | Trend | Response time per paginated request |
| `redirects_total_duration` | Trend | Total time to fetch all redirects (full pagination) |
| `redirects_requests` | Counter | Total redirects requests |
| `redirects_items_total_fetched` | Counter | Total redirect items fetched |
| `agent_hit_duration` | Trend | Response time for agent hit endpoint |
| `agent_hit_requests` | Counter | Total agent hit requests |
| `agent_post_duration` | Trend | Response time for agent post endpoint |
| `agent_post_requests` | Counter | Total agent post requests |

### Summary Statistics

The following statistics are reported for all trend metrics:

- `min` - Minimum value
- `avg` - Average value
- `med` - Median value
- `max` - Maximum value
- `p(90)` - 90th percentile
- `p(95)` - 95th percentile
- `p(99)` - 99th percentile
- `count` - Total count

## File Structure

```
loadtests/
├── load-api-agent.js        # Main test runner (all scenarios)
├── README.md                # This file
├── config/
│   ├── config.dist.json     # Config template (versioned)
│   ├── config.json          # Actual config (gitignored)
│   └── config.js            # Config loader
├── common/
│   ├── auth.js              # Authentication helpers
│   └── utils.js             # Utility functions
└── tests/
    ├── version.js           # Version endpoint test (standalone)
    ├── pages.js             # Pages endpoint test (standalone)
    ├── redirects.js         # Redirects endpoint test (standalone)
    ├── agent-hit.js         # Agent hit endpoint test (standalone)
    └── agent-post.js        # Agent post endpoint test (standalone)
```
