#!/bin/bash
# Add cron job for daily ingestion

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INGEST_BINARY="$SCRIPT_DIR/ingest"
DB_URL="postgresql://blake@localhost:5432/govcon?sslmode=disable"

# Check if binary exists
if [ ! -f "$INGEST_BINARY" ]; then
    echo "❌ Ingestion binary not found. Building it..."
    cd "$SCRIPT_DIR"
    go build -o ingest ./cmd/ingest
    if [ ! -f "$INGEST_BINARY" ]; then
        echo "❌ Failed to build binary"
        exit 1
    fi
fi

# Create log directory if it doesn't exist
LOG_DIR="$HOME/logs"
mkdir -p "$LOG_DIR"

# Get window days from environment or use default (30)
WINDOW_DAYS="${INGESTION_WINDOW_DAYS:-30}"

# Create cron entry
CRON_ENTRY="0 2 * * * cd $SCRIPT_DIR && DATABASE_URL=\"$DB_URL\" INGESTION_WINDOW_DAYS=$WINDOW_DAYS $INGEST_BINARY >> $LOG_DIR/govcon-ingest.log 2>&1"

# Add to crontab
(crontab -l 2>/dev/null | grep -v "govcon.*ingest"; echo "$CRON_ENTRY") | crontab -

echo "✅ Cron job added successfully!"
echo ""
echo "Cron entry:"
echo "  $CRON_ENTRY"
echo ""
echo "View current crontab:"
echo "  crontab -l"
echo ""
echo "View logs:"
echo "  tail -f $LOG_DIR/govcon-ingest.log"
echo ""
echo "Note: SAM API rate limit expires on Dec 25, 2025. The cron job will run daily at 2 AM after that."

