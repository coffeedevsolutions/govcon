#!/bin/bash
# Setup database and run initial ingestion

set -e

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ ERROR: DATABASE_URL environment variable is not set"
    echo ""
    echo "Please set it before running this script:"
    echo "  export DATABASE_URL='postgres://user:password@localhost:5432/dbname'"
    echo ""
    echo "Or create a .env file in the app/api directory with:"
    echo "  DATABASE_URL=postgres://user:password@localhost:5432/dbname"
    exit 1
fi

echo "✅ DATABASE_URL is set"
echo ""

# Step 1: Setup database schema
echo "📦 Step 1: Setting up database schema..."
cd "$(dirname "$0")/.."
go run ./cmd/setup-db
echo ""

# Step 2: Run initial ingestion
echo "📥 Step 2: Running initial ingestion (this may take a while)..."
go run ./cmd/ingest
echo ""

echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Set up a cron job to run ingestion daily:"
echo "   0 2 * * * cd /path/to/govcon/app/api && /path/to/go run ./cmd/ingest"
echo ""
echo "2. Or use the built binary:"
echo "   cd /path/to/govcon/app/api && go build -o ingest ./cmd/ingest"
echo "   0 2 * * * cd /path/to/govcon/app/api && ./ingest"

