---
sidebar_position: 4
---

# CLI Reference

Flecto Manager provides a command-line interface for server management, database operations, and user administration.

## Global Flags

These flags are available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to configuration file | `/etc/flecto/manager.yaml` |
| `--level` | `-l` | Log level (DEBUG, INFO, WARN, ERROR) | `INFO` |

## Commands

### start

Start the HTTP server.

```bash
flecto-manager start -c /etc/flecto/manager.yaml
```

The server will listen on the address configured in the `http.listen` configuration option.

---

### version

Display version information.

```bash
flecto-manager version
```

**Output example:**
```
flecto-manager v1.0.0 (commit: abc1234)
```

---

### validate

Validate the configuration file without starting the server.

```bash
flecto-manager validate -c /etc/flecto/manager.yaml
```

Returns exit code 0 if the configuration is valid, non-zero otherwise.

---

### db

Database management commands.

#### db init

Initialize the database with default data (admin user and role). Run this once after first installation.

```bash
flecto-manager db init -c /etc/flecto/manager.yaml
```

**Creates:**
- Admin user with username `admin` and password `admin`
- Admin role with full permissions

:::warning
Change the default admin password immediately after first login!
:::

#### db demo

Add demo data for testing purposes. This is optional and useful for development or demonstrations.

```bash
flecto-manager db demo -c /etc/flecto/manager.yaml
```

**Creates:**
- 2 namespaces (`ns1`, `ns2`)
- 6 projects (3 per namespace)
- 39 sample redirects
- 1 sample page (`robots.txt`)

#### db migrate apply

Apply pending database migrations.

```bash
# Apply all pending migrations
flecto-manager db migrate apply -c /etc/flecto/manager.yaml

# Apply a specific number of migrations
flecto-manager db migrate apply -n 2 -c /etc/flecto/manager.yaml
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--steps` | `-n` | Number of migrations to apply (0 = all) | `0` |

#### db migrate down

Rollback database migrations.

```bash
# Rollback all migrations
flecto-manager db migrate down -c /etc/flecto/manager.yaml

# Rollback a specific number of migrations
flecto-manager db migrate down -n 1 -c /etc/flecto/manager.yaml
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--steps` | `-n` | Number of migrations to rollback (0 = all) | `0` |

:::danger
Rolling back migrations may result in data loss. Always backup your database before rolling back.
:::

#### db migrate status

Show the current migration status.

```bash
flecto-manager db migrate status -c /etc/flecto/manager.yaml
```

**Output example:**
```
Migration Status
================
Current version: 20260106074436
Status: OK
```

If a migration failed and the database is in a dirty state:
```
Migration Status
================
Current version: 20260106074436
Status: DIRTY (migration failed, manual fix required)
```

---

### user

User management commands.

#### user change-password

Change a user's password from the command line.

```bash
flecto-manager user change-password -u admin -p newpassword -c /etc/flecto/manager.yaml
```

| Flag | Short | Description | Required |
|------|-------|-------------|----------|
| `--username` | `-u` | Username | Yes |
| `--password` | `-p` | New password | Yes |

:::tip
This command is useful for resetting a forgotten admin password without database access.
:::

---

## Quick Reference

```bash
# Server
flecto-manager start -c config.yaml       # Start the server
flecto-manager version                     # Show version
flecto-manager validate -c config.yaml    # Validate configuration

# Database - Initial setup
flecto-manager db migrate apply -c config.yaml  # Apply migrations
flecto-manager db init -c config.yaml           # Initialize default data
flecto-manager db demo -c config.yaml           # (Optional) Add demo data

# Database - Migrations
flecto-manager db migrate status -c config.yaml     # Show migration status
flecto-manager db migrate apply -n 1 -c config.yaml # Apply 1 migration
flecto-manager db migrate down -n 1 -c config.yaml  # Rollback 1 migration

# User management
flecto-manager user change-password -u admin -p newpass -c config.yaml
```
