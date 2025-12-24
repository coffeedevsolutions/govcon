#!/bin/bash
# Helper script to load a SAM JSON response file into the database

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <path-to-json-file>"
    echo ""
    echo "Example:"
    echo "  $0 /Users/blake/Desktop/govcon/sam-sample.json"
    exit 1
fi

JSON_FILE="$1"

if [ ! -f "$JSON_FILE" ]; then
    echo "❌ Error: File not found: $JSON_FILE"
    exit 1
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ Error: DATABASE_URL environment variable is not set"
    echo ""
    echo "Set it first:"
    echo "  export DATABASE_URL='postgresql://user@localhost:5432/dbname'"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "📄 Loading JSON file: $JSON_FILE"
echo ""

cd "$SCRIPT_DIR"
go run ./cmd/ingest-file "$JSON_FILE"

