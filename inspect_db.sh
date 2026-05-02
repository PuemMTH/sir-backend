#!/bin/bash

# Configuration
DB_NAME="sir-db"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if wrangler is available
if ! npx wrangler --version > /dev/null 2>&1; then
    echo -e "${RED}Error: wrangler is not installed or npx is not working.${NC}"
    exit 1
fi

list_tables() {
    echo -e "${BLUE}--- Tables in $DB_NAME ---${NC}"
    npx wrangler d1 execute $DB_NAME --local --command="SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';"
}

inspect_table() {
    TABLE=$1
    echo -e "${BLUE}--- Schema for table: ${YELLOW}$TABLE${BLUE} ---${NC}"
    npx wrangler d1 execute $DB_NAME --local --command="PRAGMA table_info($TABLE);"
    
    echo -e "\n${BLUE}--- Data in table: ${YELLOW}$TABLE${BLUE} (last 10 rows) ---${NC}"
    npx wrangler d1 execute $DB_NAME --local --command="SELECT * FROM $TABLE ORDER BY ROWID DESC LIMIT 10;"
}

if [ -z "$1" ]; then
    list_tables
    echo -e "\n${YELLOW}Usage:${NC} ./inspect_db.sh <table_name> to see columns and rows."
else
    inspect_table $1
fi
