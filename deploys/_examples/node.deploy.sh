#!/bin/bash
set -e

# ===========================================================
# 🚀 Node.js app (npm/pnpm/bun)
# ===========================================================

APP_DIR="/var/www/my-node-app"
BRANCH="main"
SERVICES="my-node-app"        # Supervisor services
PM="npm"                      # npm | pnpm | bun
# --------------------------------------

export PATH="/home/users/.bun/bin:/usr/local/bin:/usr/bin:$PATH"

cd "$APP_DIR"

exec 9>/tmp/$(basename "$APP_DIR")-deploy.lock
flock -n 9 || { echo "⚠️  Another deployment is already running. Skipping."; exit 0; }

echo ""
echo "======================================================"
echo "🚀 Node App Deployment: $APP_DIR ($BRANCH)"
echo "======================================================"
echo ""

echo "📡 [1/5] Contacting origin & pulling latest code..."
git fetch origin
git reset --hard "origin/$BRANCH"

echo ""
echo "📦 [2/5] Installing dependencies..."
case "$PM" in
  npm)  npm ci ;;
  pnpm) pnpm install --frozen-lockfile ;;
  bun)  bun install --frozen-lockfile ;;
esac

echo ""
echo "🏗️  [3/5] Building..."
[ -n "$(npm run 2>/dev/null | grep -E '^  build')" ] && npm run build || echo "   (no build script, skipped)"

echo ""
echo "♻️  [4/5] Respawning services..."
for SVC in $SERVICES; do
  echo "   🔄 Restarting $SVC..."
  sudo /usr/bin/supervisorctl restart "$SVC"
done

echo ""
echo "✅ [5/5] Deployment finished."
