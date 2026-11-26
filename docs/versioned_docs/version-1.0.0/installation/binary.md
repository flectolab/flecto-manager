---
sidebar_position: 2
---

# Binary Installation

Install Flecto Manager from pre-built binaries or build from source.

## Download Binary

Download the latest release from [GitHub Releases](https://github.com/flectolab/flecto-manager/releases).

### Linux (amd64)

```bash
VERSION="1.0.0"  # Replace with desired version
curl -LO "https://github.com/flectolab/flecto-manager/releases/download/${VERSION}/flecto-manager_linux_amd64.tar.gz"
tar -xzf flecto-manager_linux_amd64.tar.gz
sudo mv flecto-manager /usr/local/bin/
```

### Linux (arm64)

```bash
VERSION="1.0.0"
curl -LO "https://github.com/flectolab/flecto-manager/releases/download/${VERSION}/flecto-manager_linux_arm64.tar.gz"
tar -xzf flecto-manager_linux_arm64.tar.gz
sudo mv flecto-manager /usr/local/bin/
```

### macOS (Intel)

```bash
VERSION="1.0.0"
curl -LO "https://github.com/flectolab/flecto-manager/releases/download/${VERSION}/flecto-manager_darwin_amd64.tar.gz"
tar -xzf flecto-manager_darwin_amd64.tar.gz
sudo mv flecto-manager /usr/local/bin/
```

### macOS (Apple Silicon)

```bash
VERSION="1.0.0"
curl -LO "https://github.com/flectolab/flecto-manager/releases/download/${VERSION}/flecto-manager_darwin_arm64.tar.gz"
tar -xzf flecto-manager_darwin_arm64.tar.gz
sudo mv flecto-manager /usr/local/bin/
```

## Build from Source

Requirements:
- Go 1.24+
- Node.js 22+

```bash
# Clone repository
git clone https://github.com/flectolab/flecto-manager.git
cd flecto-manager

# Build frontend
cd webui
npm ci
npm run build
cd ..

# Build binary
go mod download
./bin/mock.sh
go tool gqlgen generate
go build -o flecto-manager .
```

## Running

### 1. Create MySQL Database

```sql
CREATE DATABASE flecto CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'flecto'@'%' IDENTIFIED BY 'your-password';
GRANT ALL PRIVILEGES ON flecto.* TO 'flecto'@'%';
FLUSH PRIVILEGES;
```

### 2. Create Configuration File

```yaml title="/etc/flecto/manager.yaml"
http:
  listen: "127.0.0.1:8080"

db:
  type: mysql
  config:
    dsn: "flecto:your-password@tcp(localhost:3306)/flecto?parseTime=true"

auth:
  jwt:
    secret: "your-32-character-secret-key-here"
```

### 3. Initialize Database

```bash
# Apply database migrations
flecto-manager db apply -c /etc/flecto/manager.yaml

# Initialize default data (admin user, etc.)
flecto-manager db init -c /etc/flecto/manager.yaml

# (Optional) Add demo data for testing
flecto-manager db demo -c /etc/flecto/manager.yaml
```

### 4. Start the Server

```bash
flecto-manager start -c /etc/flecto/manager.yaml
```

## Systemd Service

Create a systemd service for automatic startup:

```ini title="/etc/systemd/system/flecto-manager.service"
[Unit]
Description=Flecto Manager
After=network.target mysql.service

[Service]
Type=simple
User=flecto
Group=flecto
ExecStart=/usr/local/bin/flecto-manager start -c /etc/flecto/manager.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable flecto-manager
sudo systemctl start flecto-manager
```

## Upgrading

When upgrading to a new version:

```bash
# Download and install new binary
VERSION="1.1.0"
curl -LO "https://github.com/flectolab/flecto-manager/releases/download/${VERSION}/flecto-manager_linux_amd64.tar.gz"
tar -xzf flecto-manager_linux_amd64.tar.gz
sudo mv flecto-manager /usr/local/bin/

# Apply database migrations
flecto-manager db apply -c /etc/flecto/manager.yaml

# Restart the service
sudo systemctl restart flecto-manager
```

## CLI Commands

```bash
# Show version
flecto-manager version

# Start server
flecto-manager start -c config.yaml

# Database commands
flecto-manager db apply -c config.yaml   # Apply migrations
flecto-manager db init -c config.yaml    # Initialize default data
flecto-manager db demo -c config.yaml    # (Optional) Add demo data
```
