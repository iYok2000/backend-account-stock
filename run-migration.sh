#!/bin/bash
# Run migrations against production database
# Usage: export SUPABASE_DB_URL="..." then ./run-migration.sh

cd "$(dirname "$0")"

if [ -z "$SUPABASE_DB_URL" ]; then
  echo "Error: SUPABASE_DB_URL not set"
  echo "Usage: export SUPABASE_DB_URL='postgresql://...' && ./run-migration.sh"
  exit 1
fi

echo "Running migrations..."
go run ./cmd/migrate

if [ $? -eq 0 ]; then
  echo "✅ Migrations completed"
else
  echo "❌ Migration failed"
  exit 1
fi
