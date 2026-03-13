#!/bin/bash

# Supabase Management Script
# Usage: ./supabase.sh [command]

set -e

DB_PASSWORD="DUTNfUOq8wGSYHgd"
PROJECT_REF="xprbzuoydgbkeihlhxrh"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

show_help() {
    echo "🚀 Supabase Management Script"
    echo ""
    echo "Usage: ./supabase.sh [command]"
    echo ""
    echo "Commands:"
    echo "  migrate          Push migrations to Supabase"
    echo "  run              Run server with Supabase"
    echo "  create [name]    Create new migration file"
    echo "  help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./supabase.sh migrate"
    echo "  ./supabase.sh run"
    echo "  ./supabase.sh create add_new_column"
}

do_migrate() {
    echo -e "${BLUE}🔍 Syncing migrations...${NC}"
    cp migrations/*.up.sql supabase/migrations/ 2>/dev/null || true
    
    echo -e "${BLUE}🚀 Pushing to Supabase...${NC}"
    supabase db push --password "$DB_PASSWORD" --include-all
    
    echo -e "${GREEN}✅ Migration completed!${NC}"
    echo ""
    echo "💡 View tables: https://supabase.com/dashboard/project/$PROJECT_REF/editor"
}

do_run() {
    echo -e "${BLUE}🚀 Starting server with Supabase...${NC}"
    echo ""
    
    export SUPABASE_DB_URL="postgresql://postgres.$PROJECT_REF:$DB_PASSWORD@db.$PROJECT_REF.supabase.co:6543/postgres?sslmode=require"
    export APP_ENV=development
    export PORT=8080
    export JWT_SECRET=dev-secret-change-in-production
    export JWT_ISSUER=account-stock-be
    export JWT_AUDIENCE=account-stock-fe
    export ROOT_EMAIL=superadmin
    export ROOT_PASSWORD=pass@1congrate
    export ROOT_CONFIRM_CODE=YIM2021
    export CORS_ORIGIN=http://localhost:3000,http://127.0.0.1:3000
    
    echo "Press Ctrl+C to stop"
    echo ""
    
    go run ./cmd/server
}

do_create() {
    if [ -z "$1" ]; then
        echo -e "${YELLOW}❌ Please provide migration name${NC}"
        echo ""
        echo "Usage: ./supabase.sh create migration_name"
        echo "Example: ./supabase.sh create add_column_to_users"
        exit 1
    fi
    
    MIGRATION_NAME="$1"
    LAST_NUMBER=$(ls migrations/*.up.sql 2>/dev/null | tail -1 | grep -o '[0-9]\{6\}' | head -1 || echo "000000")
    NEXT_NUMBER=$(printf "%06d" $((10#$LAST_NUMBER + 1)))
    
    UP_FILE="migrations/${NEXT_NUMBER}_${MIGRATION_NAME}.up.sql"
    DOWN_FILE="migrations/${NEXT_NUMBER}_${MIGRATION_NAME}.down.sql"
    
    echo -e "${GREEN}📝 Creating migration files...${NC}"
    echo "   Up:   $UP_FILE"
    echo "   Down: $DOWN_FILE"
    
    cat > "$UP_FILE" << EOF
-- Migration: $MIGRATION_NAME
-- Created: $(date +"%Y-%m-%d %H:%M:%S")

-- Add your SQL statements below:

EOF
    
    cat > "$DOWN_FILE" << EOF
-- Rollback: $MIGRATION_NAME

-- Add your rollback SQL statements below:

EOF
    
    echo -e "${GREEN}✅ Migration files created!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Edit $UP_FILE"
    echo "  2. Add your SQL (ALTER TABLE, CREATE TABLE, etc.)"
    echo "  3. Test: go run ./cmd/migrate"
    echo "  4. Push: ./supabase.sh migrate"
}

# Main
case "${1:-help}" in
    migrate)
        do_migrate
        ;;
    run)
        do_run
        ;;
    create)
        do_create "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${YELLOW}❌ Unknown command: $1${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
