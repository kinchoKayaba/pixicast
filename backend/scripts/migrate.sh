#!/bin/bash
# Pixicast Database Migration Script
# Usage: ./scripts/migrate.sh [up|down|status]

set -e

# カラー出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 環境変数チェック
if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}ERROR: DATABASE_URL environment variable is not set${NC}"
    echo "Example: export DATABASE_URL='postgresql://user:pass@localhost:26257/pixicast?sslmode=disable'"
    exit 1
fi

# ディレクトリ設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="$BACKEND_DIR/sql/migrations"

echo -e "${GREEN}=== Pixicast Database Migration ===${NC}"
echo "Database: $DATABASE_URL"
echo "Migrations: $MIGRATIONS_DIR"
echo ""

# コマンド解析
COMMAND=${1:-up}

case $COMMAND in
    up)
        echo -e "${YELLOW}Running migrations...${NC}"
        
        # 接続確認
        echo "1. Testing database connection..."
        if psql "$DATABASE_URL" -c "SELECT version();" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Database connection OK${NC}"
        else
            echo -e "${RED}✗ Database connection failed${NC}"
            exit 1
        fi
        
        # マイグレーション実行
        echo ""
        echo "2. Running migration 001_create_tables.sql..."
        if psql "$DATABASE_URL" -f "$MIGRATIONS_DIR/001_create_tables.sql"; then
            echo -e "${GREEN}✓ Tables created${NC}"
        else
            echo -e "${RED}✗ Failed to create tables${NC}"
            exit 1
        fi
        
        echo ""
        echo "3. Running migration 002_seed_platforms.sql..."
        if psql "$DATABASE_URL" -f "$MIGRATIONS_DIR/002_seed_platforms.sql"; then
            echo -e "${GREEN}✓ Platform data seeded${NC}"
        else
            echo -e "${RED}✗ Failed to seed platforms${NC}"
            exit 1
        fi
        
        echo ""
        echo -e "${GREEN}=== Migration completed successfully! ===${NC}"
        ;;
        
    down)
        echo -e "${YELLOW}Rolling back migrations...${NC}"
        echo -e "${RED}WARNING: This will DROP all tables and data!${NC}"
        read -p "Are you sure? (yes/no): " -r
        echo
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            echo "Aborted."
            exit 0
        fi
        
        echo "Dropping tables..."
        psql "$DATABASE_URL" <<EOF
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS user_subscriptions CASCADE;
DROP TABLE IF EXISTS sources CASCADE;
DROP TABLE IF EXISTS platforms CASCADE;
EOF
        
        echo -e "${GREEN}✓ All tables dropped${NC}"
        ;;
        
    status)
        echo -e "${YELLOW}Checking migration status...${NC}"
        echo ""
        
        # テーブル一覧
        echo "=== Tables ==="
        psql "$DATABASE_URL" -c "\dt" || true
        
        echo ""
        echo "=== Platforms ==="
        psql "$DATABASE_URL" -c "SELECT * FROM platforms;" 2>/dev/null || echo "platforms table does not exist"
        
        echo ""
        echo "=== Sources Count ==="
        psql "$DATABASE_URL" -c "SELECT COUNT(*) as sources_count FROM sources;" 2>/dev/null || echo "sources table does not exist"
        
        echo ""
        echo "=== Events Count ==="
        psql "$DATABASE_URL" -c "SELECT COUNT(*) as events_count FROM events;" 2>/dev/null || echo "events table does not exist"
        
        echo ""
        echo "=== Subscriptions Count ==="
        psql "$DATABASE_URL" -c "SELECT COUNT(*) as subscriptions_count FROM user_subscriptions;" 2>/dev/null || echo "user_subscriptions table does not exist"
        ;;
        
    *)
        echo "Usage: $0 [up|down|status]"
        echo ""
        echo "Commands:"
        echo "  up      - Run migrations (create tables and seed data)"
        echo "  down    - Rollback migrations (drop all tables)"
        echo "  status  - Show current migration status"
        exit 1
        ;;
esac

