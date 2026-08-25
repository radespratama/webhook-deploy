#!/bin/bash
set -e

# ===========================================================
# 🚀 Docker Compose app
# ===========================================================

APP_DIR="/var/www/my-docker-app"
BRANCH="main"
COMPOSE="docker compose"      # lama: "docker-compose"
# --------------------------------------

export PATH="/usr/local/bin:/usr/bin:$PATH"

cd "$APP_DIR"

exec 9>/tmp/$(basename "$APP_DIR")-deploy.lock
flock -n 9 || { echo "⚠️  Another deployment is already running. Skipping."; exit 0; }

echo ""
echo "======================================================"
echo "🚀 Docker Deployment: $APP_DIR ($BRANCH)"
echo "======================================================"
echo ""

echo "📡 [1/4] Contacting origin & pulling latest code..."
git fetch origin
git reset --hard "origin/$BRANCH"

echo ""
echo "🏗️  [2/4] Building images..."
$COMPOSE build

echo ""
echo "♻️  [3/4] Recreating containers..."
$COMPOSE up -d --remove-orphans

echo ""
echo "🧹 [4/4] Pruning old images..."
docker image prune -f

echo ""
echo "✅ Deployment finished."
