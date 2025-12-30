#!/bin/bash
# Migration runner script that ensures psql is in PATH

# Try to find psql in common Homebrew locations
if ! command -v psql &> /dev/null; then
    # Check common Homebrew PostgreSQL locations
    for path in \
        "/opt/homebrew/opt/postgresql@16/bin/psql" \
        "/opt/homebrew/opt/postgresql/bin/psql" \
        "/usr/local/Cellar/postgresql@16/16.11/bin/psql" \
        "/usr/local/Cellar/postgresql@16/16.10/bin/psql" \
        "/usr/local/Cellar/postgresql@16/16.9/bin/psql" \
        "/usr/local/opt/postgresql@16/bin/psql" \
        "/usr/local/opt/postgresql/bin/psql" \
        "/usr/local/bin/psql"
    do
        if [ -f "$path" ]; then
            export PATH="$(dirname "$path"):$PATH"
            break
        fi
    done
fi

# Check if psql is now available
if ! command -v psql &> /dev/null; then
    echo "Error: psql not found. Please install PostgreSQL or add it to your PATH."
    echo "On macOS with Homebrew: brew install postgresql@16"
    exit 1
fi

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Accept migration file as argument, or default to latest
MIGRATION_FILE="${1:-$API_DIR/migrations/003_opportunity_description.sql}"

# If no argument provided and default doesn't exist, try 002
if [ ! -f "$MIGRATION_FILE" ] && [ -z "$1" ]; then
    MIGRATION_FILE="$API_DIR/migrations/002_search_indexes.sql"
fi

# Check if migration file exists
if [ ! -f "$MIGRATION_FILE" ]; then
    echo "Error: Migration file not found: $MIGRATION_FILE"
    exit 1
fi

# Load .env file from repo root
REPO_ROOT="$(cd "$API_DIR/../.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"

if [ ! -f "$ENV_FILE" ]; then
    echo "Warning: .env file not found at $ENV_FILE"
    echo "Using DATABASE_URL from environment..."
else
    # Source .env file
    set -a
    source "$ENV_FILE"
    set +a
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL is not set"
    exit 1
fi

# Run the migration
echo "Running migration: $MIGRATION_FILE"
psql "$DATABASE_URL" -f "$MIGRATION_FILE"

exit $?

