# Configuring Ingestion Window Size

The ingestion job can be configured to use a smaller date range to reduce the number of API calls and potentially avoid rate limits.

## Default Behavior

By default, the ingestion uses a **30-day rolling window**, pulling opportunities from the last 30 days.

## Configuring Window Size

Set the `INGESTION_WINDOW_DAYS` environment variable to change the window size:

```bash
# Use 7-day window
INGESTION_WINDOW_DAYS=7 go run ./cmd/ingest

# Use 14-day window
INGESTION_WINDOW_DAYS=14 go run ./cmd/ingest
```

## Trade-offs

**Smaller window (e.g., 7 days):**
- ✅ Fewer API calls per run
- ✅ Faster ingestion
- ✅ Less likely to hit rate limits
- ❌ May miss updates to older opportunities
- ❌ Requires more frequent runs to maintain coverage

**Larger window (e.g., 30 days):**
- ✅ Captures updates to opportunities over a longer period
- ✅ More comprehensive coverage
- ❌ More API calls per run
- ❌ Slower ingestion
- ❌ Higher risk of hitting rate limits

## Recommended Settings

- **Daily cron job**: Use 30 days (default) for comprehensive coverage
- **If hitting rate limits**: Reduce to 14-21 days
- **For testing/development**: Use 7 days for faster runs

## Updating Cron Job

To change the window size in your cron job, update the script or edit crontab directly:

```bash
# Edit crontab
crontab -e

# Update the line to include INGESTION_WINDOW_DAYS
0 2 * * * cd /path/to/govcon/app/api && DATABASE_URL="..." INGESTION_WINDOW_DAYS=14 ./ingest >> /var/log/govcon-ingest.log 2>&1
```

## Current Rate Limit Status

⚠️ **Note**: As of Dec 23, 2025, the SAM API rate limit has been exceeded. Access will be restored on Dec 25, 2025.

Even with a smaller window, you'll still hit the rate limit until the quota resets. Once it resets, using a smaller window can help you stay within limits.

