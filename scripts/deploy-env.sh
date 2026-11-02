#!/usr/bin/env bash
set -euo pipefail

# Alpacard Multi-Environment Deployment Script
# Usage: ./scripts/deploy-env.sh [dev|prod] [IMAGE_TAG]

TARGET_ENV="${1:-dev}"
IMAGE_TAG="${2:-latest}"

if [ "$TARGET_ENV" = "prod" ] || [ "$TARGET_ENV" = "production" ] || [ "$TARGET_ENV" = "main" ]; then
  NAMESPACE="alpacard"
  DOMAIN="https://alpacard.app"
  ENV_LABEL="Production"
elif [ "$TARGET_ENV" = "dev" ] || [ "$TARGET_ENV" = "development" ]; then
  NAMESPACE="alpacard-dev"
  DOMAIN="https://dev.alpacard.app"
  ENV_LABEL="Development"
else
  echo "❌ Error: Invalid target environment '$TARGET_ENV'. Must be 'dev' or 'prod'."
  exit 1
fi

echo "=========================================================="
echo "🚀 Alpacard Continuous Deployment -> $ENV_LABEL ($NAMESPACE)"
echo "   Target Domain: $DOMAIN"
echo "   Timestamp:     $(date -u)"
echo "=========================================================="

# 1. Build Shared Types and Frontend Bundles
echo "📦 [1/6] Installing dependencies and building frontend apps..."
pnpm install --frozen-lockfile
pnpm --filter @alpacard/shared build
pnpm --filter @alpacard/web build
pnpm --filter @alpacard/wallet build

# 2. Compile Go Binaries Statically
echo "🐹 [2/6] Compiling Go API & Worker binaries statically..."
(cd apps/api && CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/server)
(cd apps/api && CGO_ENABLED=0 GOOS=linux go build -o bin/worker ./cmd/worker)

# 3. Build Container Images with Podman
echo "🐳 [3/6] Building container images..."
podman build -t "ghcr.io/alpacard/alpacard-web:${IMAGE_TAG}" apps/web
podman build -t "ghcr.io/alpacard/alpacard-wallet:${IMAGE_TAG}" apps/wallet
podman build -t "ghcr.io/alpacard/alpacard-api:${IMAGE_TAG}" apps/api

# 4. Import Container Images into Kubernetes containerd
echo "📥 [4/6] Importing container images into Kubernetes containerd runtime..."
podman save "ghcr.io/alpacard/alpacard-web:${IMAGE_TAG}" | sudo k3s ctr --namespace k8s.io images import -
podman save "ghcr.io/alpacard/alpacard-wallet:${IMAGE_TAG}" | sudo k3s ctr --namespace k8s.io images import -
podman save "ghcr.io/alpacard/alpacard-api:${IMAGE_TAG}" | sudo k3s ctr --namespace k8s.io images import -

# 5. Rotate Running Pods in the targeted Namespace
echo "🔄 [5/6] Applying image updates and triggering rollout for namespace: $NAMESPACE..."
kubectl set image deployment/alpacard-api api="ghcr.io/alpacard/alpacard-api:${IMAGE_TAG}" -n "$NAMESPACE"
kubectl set image deployment/alpacard-worker worker="ghcr.io/alpacard/alpacard-api:${IMAGE_TAG}" -n "$NAMESPACE"
kubectl set image deployment/alpacard-web web="ghcr.io/alpacard/alpacard-web:${IMAGE_TAG}" -n "$NAMESPACE"
kubectl set image deployment/alpacard-wallet wallet="ghcr.io/alpacard/alpacard-wallet:${IMAGE_TAG}" -n "$NAMESPACE"

echo "⏳ Awaiting rollout completion..."
kubectl rollout status deployment/alpacard-web -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/alpacard-wallet -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/alpacard-api -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/alpacard-worker -n "$NAMESPACE" --timeout=120s

# 6. Automated Health Verification Smoke Test
echo "🔍 [6/6] Verifying environment health at $DOMAIN..."
STATUS=$(curl -s -k -o /dev/null -w "%{http_code}" "$DOMAIN/api/health" || echo "000")
if [ "$STATUS" = "200" ]; then
  echo "✅ Health check PASSED (HTTP 200) for $DOMAIN/api/health"
else
  echo "⚠️ Warning: Health check returned HTTP $STATUS. Inspecting endpoint..."
  curl -s -k "$DOMAIN/api/health" || true
fi

echo "=========================================================="
echo "🎉 Deployment to $ENV_LABEL successfully completed!"
echo "=========================================================="
