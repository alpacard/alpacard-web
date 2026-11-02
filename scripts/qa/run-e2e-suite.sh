#!/usr/bin/env bash
set -e

# ==============================================================================
# Alpacard QA Automation - Full E2E & Database Verification Suite
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

API_BASE_URL="${1:-${BASE_URL:-https://dev.alpacard.app}}"
DB_TARGET_URL="${2:-${DATABASE_URL:-postgres://alpacard_dev:alpacard_dev_secret_2026@dev-db.alpacard.app:5432/alpacard_dev?sslmode=disable}}"

echo ""
echo "=================================================================="
echo "🎯 STARTING ALPACARD QA AUTOMATED TEST SUITE (API + DATABASE)"
echo "=================================================================="
echo "🌐 API Target:       $API_BASE_URL"
echo "🗄️ Database Target:  dev-db.alpacard.app:5432 (or direct IP 192.168.0.54)"
echo "=================================================================="
echo ""

# Step 1: Run Newman Postman E2E API Suite
echo "📡 [1/2] Executing Automated API Journey Tests (Newman)..."
cd "$ROOT_DIR"
npx --yes newman run alpacard-postman-collection.json \
  --env-var "baseUrl=$API_BASE_URL" \
  --reporters cli

echo ""
echo "🗄️ [2/2] Executing Direct Relational Database Integrity Verification..."
cd "$ROOT_DIR/apps/api"
DATABASE_URL="$DB_TARGET_URL" go run ./cmd/qa-test-db/main.go

echo "=================================================================="
echo "✅ QA AUTOMATION TEST SUITE RUN COMPLETED SUCCESSFULLY!"
echo "=================================================================="
