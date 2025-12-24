#!/bin/bash
# Helper script to set up cron job for daily ingestion

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INGEST_PATH="$SCRIPT_DIR/cmd/ingest"

echo "🔧 Setting up cron job for daily ingestion"
echo ""

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "⚠️  WARNING: DATABASE_URL is not set in current environment"
    echo "   You'll need to set it in the cron job or use a .env file"
    echo ""
fi

# Build the binary
echo "📦 Building ingestion binary..."
cd "$SCRIPT_DIR"
go build -o ingest ./cmd/ingest

if [ ! -f "$SCRIPT_DIR/ingest" ]; then
    echo "❌ Failed to build binary"
    exit 1
fi

echo "✅ Binary built: $SCRIPT_DIR/ingest"
echo ""

# Get absolute path
INGEST_BINARY="$(cd "$SCRIPT_DIR" && pwd)/ingest"

# Create cron entry
CRON_TIME="0 2"  # 2 AM daily
CRON_CMD="cd $SCRIPT_DIR && DATABASE_URL=\"\$DATABASE_URL\" $INGEST_BINARY >> /var/log/govcon-ingest.log 2>&1"

echo "📝 Cron job configuration:"
echo "   Time: Daily at 2:00 AM"
echo "   Command: $CRON_CMD"
echo ""
echo "To add this to your crontab, run:"
echo ""
echo "crontab -e"
echo ""
echo "Then add this line:"
echo ""
if [ -n "$DATABASE_URL" ]; then
    echo "0 2 * * * cd $SCRIPT_DIR && DATABASE_URL=\"$DATABASE_URL\" $INGEST_BINARY >> /var/log/govcon-ingest.log 2>&1"
else
    echo "0 2 * * * cd $SCRIPT_DIR && DATABASE_URL=\"your-database-url-here\" $INGEST_BINARY >> /var/log/govcon-ingest.log 2>&1"
fi
echo ""
echo "Or use this one-liner to add it automatically:"
echo ""
if [ -n "$DATABASE_URL" ]; then
    echo "(crontab -l 2>/dev/null; echo \"0 2 * * * cd $SCRIPT_DIR && DATABASE_URL=\\\"$DATABASE_URL\\\" $INGEST_BINARY >> /var/log/govcon-ingest.log 2>&1\") | crontab -"
else
    echo "# Set DATABASE_URL first, then run:"
    echo "(crontab -l 2>/dev/null; echo \"0 2 * * * cd $SCRIPT_DIR && DATABASE_URL=\\\"\$DATABASE_URL\\\" $INGEST_BINARY >> /var/log/govcon-ingest.log 2>&1\") | crontab -"
fi

