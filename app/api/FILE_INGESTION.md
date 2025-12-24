# Loading JSON Files into Database

You can load SAM.gov API response JSON files directly into the database using the file ingestion tool.

## Quick Start

```bash
# Set your database URL
export DATABASE_URL="postgresql://blake@localhost:5432/govcon?sslmode=disable"

# Load a JSON file
./scripts/load-json.sh /path/to/sam-response.json

# Or use go run directly
go run ./cmd/ingest-file /path/to/sam-response.json
```

## JSON File Format

The JSON file should match the SAM.gov API response format:

```json
{
  "totalRecords": 2471,
  "limit": 10,
  "offset": 0,
  "opportunitiesData": [
    {
      "noticeId": "...",
      "title": "...",
      "postedDate": "...",
      ...
    }
  ]
}
```

## What Happens

1. The tool reads the JSON file
2. Parses each opportunity in `opportunitiesData`
3. Computes content hash for change detection
4. Inserts new opportunities or updates existing ones
5. Logs statistics (new, updated, skipped, errors)

## Behavior

- **New opportunities**: Inserted into both `opportunity_raw` and `opportunity` tables
- **Existing opportunities with changes**: Updated in both tables, version logged
- **Existing opportunities without changes**: Skipped (no database writes)

This ensures idempotency - you can safely load the same file multiple times.

## Example

```bash
# Load your sample file
export DATABASE_URL="postgresql://blake@localhost:5432/govcon?sslmode=disable"
./scripts/load-json.sh ~/Desktop/govcon/sam-sample.json
```

Output:
```
✅ Acquired advisory lock, starting file ingestion...
📄 Loaded 10 opportunities from file
✅ File ingestion completed successfully
📊 Statistics:
   Total processed: 10
   New: 0
   Updated: 0
   Skipped: 10
   Errors: 0
```

## Use Cases

- **Testing**: Load sample data for development
- **Backfilling**: Load historical data from saved API responses
- **Recovery**: Re-load data if there were ingestion errors
- **Development**: Populate database without hitting API rate limits

## Notes

- Uses the same advisory lock as the regular ingestion job
- If another ingestion is running, this will exit gracefully
- All opportunities are processed with the same change detection logic
- Raw JSON is stored for audit and reprocessing

