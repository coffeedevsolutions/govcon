# Quick Start Guide

## Prerequisites

1. PostgreSQL database running
2. `DATABASE_URL` environment variable set

## Step-by-Step Setup

### Step 1: Set Database URL

```bash
export DATABASE_URL="postgres://username:password@localhost:5432/dbname"
```

**Common examples:**
- Local PostgreSQL: `postgres://postgres:postgres@localhost:5432/govcon`
- Docker: `postgres://postgres:postgres@localhost:5432/govcon`
- Cloud (e.g., Supabase): `postgres://user:pass@host:5432/dbname`

### Step 2: Create Database (if needed)

```bash
# Using psql
createdb govcon

# Or using Docker
docker run --name postgres-govcon -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=govcon -p 5432:5432 -d postgres
```

### Step 3: Run Database Setup

```bash
cd app/api
go run ./cmd/setup-db
```

Expected output:
```
✅ Created ping table
✅ Enabled pg_trgm extension
✅ Created opportunity_raw table
✅ Created opportunity table
✅ Created GIN full-text search index
✅ Created pg_trgm indexes for fuzzy matching
✅ Created opportunity_version table
✅ Database setup complete!
```

### Step 4: Run Initial Ingestion

```bash
go run ./cmd/ingest
```

This will:
- Pull opportunities from the last 30 days
- Take several minutes depending on data volume
- Show statistics when complete

Expected output:
```
✅ Acquired advisory lock, starting ingestion...
📅 Pulling opportunities from 11/23/2024 to 12/23/2024
✅ Ingestion completed successfully
📊 Statistics:
   Total processed: 1234
   New: 500
   Updated: 50
   Skipped: 684
   Errors: 0
```

### Step 5: Set Up Daily Cron Job

**Option A: Use the helper script**
```bash
./scripts/setup-cron.sh
```

**Option B: Manual setup**
```bash
# Build binary
go build -o ingest ./cmd/ingest

# Add to crontab
crontab -e

# Add this line (adjust DATABASE_URL and path):
0 2 * * * cd /Users/blake/Desktop/govcon/app/api && DATABASE_URL="your-url" ./ingest >> /var/log/govcon-ingest.log 2>&1
```

### Step 6: Start API Server

```bash
go run ./cmd/api
```

The API will be available at `http://localhost:4000`

Test it:
```bash
curl http://localhost:4000/health
curl http://localhost:4000/opportunities?limit=5
```

## Troubleshooting

### "DATABASE_URL is not set"
- Make sure you've exported the environment variable
- Or create a `.env` file (you may need to load it manually)

### "Failed to connect to database"
- Check that PostgreSQL is running
- Verify DATABASE_URL is correct
- Test connection: `psql $DATABASE_URL`

### "Another ingestion job is already running"
- This is normal if a job is already running
- The new job exits gracefully (code 0)
- Wait for the current job to finish

### Ingestion is slow
- This is normal for the first run (30 days of data)
- Subsequent runs will be faster (only new/updated records)

