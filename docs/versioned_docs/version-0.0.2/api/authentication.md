---
sidebar_position: 1
---

# Authentication

The Flecto Manager API supports two authentication methods: JWT tokens obtained via login, and API tokens generated in the admin interface.

## Authentication Methods

### 1. JWT Authentication (User Login)

For interactive use or scripts that need user-level access.

#### Login

```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "your-password"
}
```

**Response:**

```json
{
  "user": {
    "id": 1,
    "username": "admin",
    "firstname": "Admin",
    "lastname": "User"
  },
  "tokens": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresAt": 1704067200
  }
}
```

#### Using the Access Token

Include the access token in the `Authorization` header:

```http
GET /api/namespace/production/project/my-website/version
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

#### Refresh Token

When the access token expires, use the refresh token to get a new one:

```http
POST /auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Logout

Invalidate the refresh token:

```http
POST /auth/logout
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### 2. API Token (Recommended for Agents)

For agents and automated systems, generate an API token in the admin interface. API tokens don't expire (unless configured) and can have specific permissions.

#### Creating an API Token

1. Login to the Manager dashboard
2. Go to **Settings** → **API Tokens**
3. Click **Create Token**
4. Configure:
   - **Name**: Descriptive name (e.g., `traefik-production`)
   - **Expiration**: Optional expiration date
   - **Permissions**: Select the resources and actions allowed
5. Copy the token immediately (it won't be shown again)

#### Permissions for Agents

Agents need specific permissions to function properly:

| Resource | Permission | Purpose |
|----------|------------|---------|
| Redirect | Read | Fetch redirect rules |
| Page | Read | Fetch static pages |
| Agent | Write | Update agent status and heartbeat |

The **Agent Write** permission is required for agents to report their status (sync success/error, version, load duration) back to the Manager.

#### Token Format

API tokens have the format: `flecto_xxxxxxxxxxxx`

#### Using an API Token

Include the token in the `Authorization` header:

```http
GET /api/namespace/production/project/my-website/redirects
Authorization: Bearer flecto_abcd1234efgh5678
```

## Comparison

| Feature | JWT (Login) | API Token |
|---------|-------------|-----------|
| Obtained via | `/auth/login` endpoint | Admin dashboard |
| Expires | Yes (configurable, default 15min) | Optional |
| Refresh needed | Yes | No |
| Permissions | Based on user roles | Configured per token |
| Best for | Interactive use, admin scripts | Agents, CI/CD, automation |

## Error Responses

### Invalid Credentials

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "invalid_credentials",
  "message": "Invalid email or password"
}
```

### User Not Found

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json

{
  "error": "user_not_exist",
  "message": "User account not exist"
}
```

### Missing or Invalid Token

```http
HTTP/1.1 401 Unauthorized
```

### Insufficient Permissions

```http
HTTP/1.1 403 Forbidden
```

## OpenID Connect

If OpenID Connect is enabled, you can also authenticate via your identity provider.

### Check OpenID Configuration

```http
GET /auth/openid
```

**Response (enabled):**

```json
{
  "enabled": true,
  "authUrl": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

**Response (disabled):**

```json
{
  "enabled": false
}
```

### Login Flow

1. Redirect user to the `authUrl`
2. User authenticates with the identity provider
3. Provider redirects to `/auth/openid/callback`
4. Callback returns JWT tokens
