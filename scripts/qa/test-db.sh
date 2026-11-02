#!/usr/bin/env bash
set -e

# Alpacard QA Automation Database Verification Runner
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🚀 Running Alpacard QA Database Verification Tool..."
cd "$ROOT_DIR/apps/api"
go run ./cmd/qa-test-db/main.go "$@"
