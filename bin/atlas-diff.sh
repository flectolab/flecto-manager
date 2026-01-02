#!/bin/bash
set -e

# Script to generate migration diff using Atlas
# Usage: ./bin/atlas-diff.sh <migration_name>
#
# Prerequisites:
#   - Atlas CLI installed: curl -sSf https://atlasgo.sh | sh
#   - Docker compose running: docker compose up -d db
#   - Go dependencies: go mod download
MIGRATION_PATH="migrations"
MIGRATION_NAME=${1:-"migration"}

if [ -z "$1" ]; then
    echo "Usage: $0 <migration_name>"
    echo "Example: $0 add_user_fields"
    exit 1
fi

# Check if atlas is installed
if ! command -v atlas &> /dev/null; then
    echo "Error: Atlas CLI is not installed"
    echo "Install it with: curl -sSf https://atlasgo.sh | sh"
    exit 1
fi

# Check if DB is running
DB_PORT=${FLECTO_DB_PORT:-3306}
if ! nc -z localhost "$DB_PORT" 2>/dev/null; then
    echo "Error: Database is not running on port $DB_PORT"
    echo "Start it with: docker compose up -d db"
    exit 1
fi

# Create migrations directory if not exists
mkdir -p ${MIGRATION_PATH}

# Atlas config file location
ATLAS_CONFIG="file://./tools/atlas-loader/atlas.hcl"

# Generate the migration diff
echo "Generating migration diff: $MIGRATION_NAME"
atlas migrate diff "$MIGRATION_NAME" --env local --config "$ATLAS_CONFIG"

echo ""
echo "Migration generated in ${MIGRATION_PATH}"
echo ""
echo "Next steps:"
echo "  1. Review and edit the generated SQL file"
echo "  2. After editing, recalculate the checksum:"
echo "     atlas migrate hash --env local --config $ATLAS_CONFIG"
echo "  3. Apply with: go run main.go db migrate up"
