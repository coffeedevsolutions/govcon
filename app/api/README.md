# Government Opportunities API

This API provides access to government contracting opportunities from SAM.gov.

## Setup

### 1. Database Configuration

Set the `DATABASE_URL` environment variable:

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/govcon"
```

Or create a `.env` file in this directory:
```
DATABASE_URL=postgres://user:password@localhost:5432/govcon
```

### 2. Database Schema Setup

Run the setup script to create all necessary tables and indexes:

```bash
go run ./cmd/setup-db
```

This will create:
- `opportunity_raw` - Raw JSON snapshots
- `opportunity` - Normalized opportunity data
- `opportunity_version` - Change history
- All necessary indexes (GIN, pg_trgm, etc.)

### 2a. Run Search Indexes Migration

After initial setup, run the search indexes migration to add search columns and indexes:

```bash
pnpm --filter api db:migrate
```

Or run the migration script directly:
```bash
bash app/api/scripts/run-migration.sh
```

Or manually (if psql is in your PATH):
```bash
psql "$DATABASE_URL" -f app/api/migrations/002_search_indexes.sql
```

**Note:** The `pnpm db:migrate` command uses a script that automatically finds `psql` in common Homebrew locations, so you don't need to have it in your PATH.

This migration will:
- Add `solicitation_number` and `agency_path_name` columns
- Backfill data from `opportunity_raw.raw_data`
- Create full-text search index (`search_tsv` tsvector column)
- Add filter indexes (NAICS, set-aside, state, agency, dates)
- Add trigram indexes for fuzzy matching

### 3. Initial Ingestion

Run the ingestion job to populate the database:

```bash
go run ./cmd/ingest
```

This will:
- Pull opportunities from the last 30 days
- Store raw and normalized data
- Track changes via content hashing
- Log statistics

### 4. Daily Ingestion (Cron Job)

Set up a daily cron job to keep data fresh. You can either:

**Option A: Use the helper script**
```bash
./scripts/setup-cron.sh
```

**Option B: Manual cron setup**

Build the binary:
```bash
go build -o ingest ./cmd/ingest
```

Add to crontab:
```bash
crontab -e
```

Add this line (runs daily at 2 AM):
```
0 2 * * * cd /path/to/govcon/app/api && DATABASE_URL="your-db-url" ./ingest >> /var/log/govcon-ingest.log 2>&1
```

**Option C: Use go run (simpler but slower)**
```
0 2 * * * cd /path/to/govcon/app/api && DATABASE_URL="your-db-url" go run ./cmd/ingest >> /var/log/govcon-ingest.log 2>&1
```

## Running the API Server

```bash
go run ./cmd/api
```

The API will run on `http://localhost:4000`

### API Endpoints

- `GET /health` - Health check
- `GET /opportunities` - Search opportunities (legacy endpoint with OFFSET pagination)
  - Query parameters:
    - `postedFrom` - Start date (MM/DD/YYYY)
    - `postedTo` - End date (MM/DD/YYYY)
    - `active` - Filter by active status (true/false)
    - `ptype` - Opportunity type
    - `search` - Full-text search query
    - `limit` - Results per page (default: 10)
    - `offset` - Pagination offset (default: 0)

- `GET /opportunities/search` - Fast search with keyset pagination (recommended)
  - Query parameters (all optional):
    - `q` - Keyword search (searches title, solicitation number, agency, description)
    - `naics` - NAICS code (exact match)
    - `setAside` - Set-aside type (exact match, e.g., "SBA")
    - `state` - State code (exact match, e.g., "MO")
    - `agency` - Agency name (prefix match)
    - `postedFrom` - Posted date from (YYYY-MM-DD or MM/DD/YYYY)
    - `postedTo` - Posted date to (YYYY-MM-DD or MM/DD/YYYY)
    - `dueFrom` - Response deadline from (YYYY-MM-DD or MM/DD/YYYY)
    - `dueTo` - Response deadline to (YYYY-MM-DD or MM/DD/YYYY)
    - `sort` - Sort order: `posted_desc` (default), `due_asc`, `relevance`
    - `limit` - Results per page (default: 25, max: 100)
    - `cursor` - Keyset pagination cursor (from previous response)
  - Response:
    ```json
    {
      "items": [...],
      "nextCursor": "base64encoded..." | null,
      "debug": { "sort": "...", "appliedFilters": {...} }
    }
    ```

- `GET /opportunities/:noticeId` - Get individual opportunity by notice ID

## Architecture

- **Ingestion**: Daily cron job pulls from SAM.gov with 30-day rolling window
- **Storage**: Raw JSON + normalized columns for fast querying
- **Change Detection**: SHA256 hash-based change detection
- **Search**: Postgres GIN full-text search + pg_trgm fuzzy matching
- **Locking**: Postgres advisory locks prevent concurrent ingestion jobs

## Testing the Search Endpoint

### Example curl requests:

```bash
# Basic search
curl "http://localhost:4000/opportunities/search?q=wire&limit=10"

# Filter by NAICS and state
curl "http://localhost:4000/opportunities/search?naics=335311&state=MO&sort=posted_desc"

# Date range with keyword search
curl "http://localhost:4000/opportunities/search?q=software&postedFrom=2025-12-01&postedTo=2025-12-31"

# Pagination (use cursor from previous response)
curl "http://localhost:4000/opportunities/search?cursor=eyJwb3N0ZWREYXRlIjoiMjAyNS0xMi0yMyIsIm5vdGljZUlkIjoiYWJjMTIzIn0="

# Multiple filters combined
curl "http://localhost:4000/opportunities/search?naics=561730&setAside=SBA&state=OK&agency=DEPT%20OF%20DEFENSE&sort=due_asc"
```

### Example filter URLs for frontend:

- Keyword search: `/opportunities?q=wire`
- NAICS filter: `/opportunities?naics=335311`
- State filter: `/opportunities?state=MO`
- Combined filters: `/opportunities?q=software&naics=541511&state=CA&sort=relevance`
- Date range: `/opportunities?postedFrom=2025-12-01&postedTo=2025-12-31`

### Performance Notes

- All queries use parameterized SQL for safety
- Keyset pagination avoids OFFSET performance issues
- Full-text search uses GIN index on `search_tsv` column
- Filter indexes ensure fast queries even with multiple filters
- Check index usage with: `EXPLAIN ANALYZE SELECT ...`

## Development

### Build all binaries:
```bash
go build ./cmd/api
go build ./cmd/ingest
go build ./cmd/setup-db
```

### Run tests:
```bash
go test ./...
```

