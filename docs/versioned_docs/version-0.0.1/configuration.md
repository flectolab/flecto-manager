---
sidebar_position: 3
---

# Configuration

Flecto Manager is configured via a YAML file. By default, it looks for `/etc/flecto/manager.yaml`.

## Full Configuration Reference

```yaml
# HTTP server configuration
http:
  listen: "127.0.0.1:8080"  # Address to bind

# Database configuration
db:
  type: mysql  # Database type (mysql)
  config:
    dsn: "user:password@tcp(localhost:3306)/flecto?parseTime=true"

# Authentication configuration
auth:
  jwt:
    secret: "minimum-32-characters-secret-key"  # JWT signing secret
    issuer: "flecto-manager"                     # JWT issuer claim
    access_token_ttl: 15m                        # Access token lifetime
    refresh_token_ttl: 24h                       # Refresh token lifetime
    header_name: "Authorization"                 # Header for JWT token

  openid:
    enabled: false           # Enable OpenID Connect
    provider_url: ""         # OIDC provider base URL (without /.well-known/openid-configuration)
    client_id: ""            # OIDC client ID
    client_secret: ""        # OIDC client secret
    redirect_url: ""         # Callback URL
    roles_claim: ""          # JWT claim containing user roles (optional)

# Page limits
page:
  size_limit: 1048576        # Max size per page (1MB)
  total_size_limit: 104857600 # Max total size (100MB)

# Agent configuration
agent:
  offline_threshold: 6h      # Mark agent offline after this duration
```

## Environment Variables

When running in Docker, you can configure via environment variables:

| Variable | Description |
|----------|-------------|
| `FLECTO_MANAGER_CFG` | Full YAML configuration content |
| `FLECTO_MANAGER_CONFIG_PATH` | Path to configuration file |
| `LOG_LEVEL` | Log level: `DEBUG`, `INFO`, `WARN`, `ERROR` |

### Example with Environment Variable

```bash
docker run -d \
  -e FLECTO_MANAGER_CFG='
http:
  listen: "0.0.0.0:8080"
db:
  type: mysql
  config:
    dsn: "user:password@tcp(mysql:3306)/flecto?parseTime=true"
auth:
  jwt:
    secret: "your-32-character-secret-key-here"
' \
  ghcr.io/flectolab/flecto-manager:1.0.0
```

## Database

Flecto Manager uses MySQL as its database.

### MySQL DSN Format

```
user:password@tcp(host:port)/database?parseTime=true
```

Example:
```yaml
db:
  type: mysql
  config:
    dsn: "flecto:secretpassword@tcp(127.0.0.1:3306)/flecto_manager?parseTime=true"
```

### Database Commands

```bash
# Apply migrations (run after installation and after each upgrade)
flecto-manager db apply

# Initialize default data (run once after first installation)
flecto-manager db init

# (Optional) Add demo data for testing
flecto-manager db demo
```

## OpenID Connect

To enable SSO with an OpenID Connect provider:

```yaml
auth:
  openid:
    enabled: true
    provider_url: "https://accounts.google.com"
    client_id: "your-client-id"
    client_secret: "your-client-secret"
    redirect_url: "https://flecto.example.com/auth/callback"
    roles_claim: "groups"  # Optional: map roles from OIDC token
```

:::info
The `provider_url` should be the base URL of the OIDC provider. The `/.well-known/openid-configuration` endpoint is automatically appended when discovering the provider configuration.
:::

### Role Mapping with `roles_claim`

The `roles_claim` option allows automatic role assignment based on claims from the OIDC token. When configured, Flecto Manager reads the specified claim from the ID token and assigns matching roles to the user.

For example, if your identity provider returns:

```json
{
  "sub": "user123",
  "email": "user@example.com",
  "groups": ["flecto-admin", "flecto-editor"]
}
```

With `roles_claim: "groups"`, the user will be assigned the `flecto-admin` and `flecto-editor` roles (if they exist in Flecto Manager).

:::tip
Create roles in Flecto Manager with names matching your identity provider's groups/roles to enable automatic mapping.
:::

### Examples

| Provider | provider_url |
|----------|--------------|
| Google | `https://accounts.google.com` |
| Keycloak | `https://keycloak.example.com/realms/myrealm` |
| Auth0 | `https://your-tenant.auth0.com` |
| Azure AD | `https://login.microsoftonline.com/{tenant-id}/v2.0` |

Supported providers include:
- Google
- Keycloak
- Auth0
- Azure AD
- Any OIDC-compliant provider

## Security Recommendations

1. **Use a strong JWT secret** - At least 32 characters, randomly generated
2. **Use HTTPS** - Always run behind a reverse proxy with TLS
3. **Change default password** - Change `admin` password immediately
4. **Restrict network access** - Bind to localhost if using a reverse proxy
