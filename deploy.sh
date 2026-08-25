#!/bin/bash
set -e  

# ===========================================================
# 🚀 Engine Deployment Script
# ===========================================================

APP_DIR="/var/www/engine"
BRANCH="main"                 # Branch target
SERVICES="api web"  # Supervisor services
# --------------------------------------

export PATH="/home/users/.bun/bin:/usr/local/bin:/usr/bin:$PATH"

cd "$APP_DIR"

exec 9>/tmp/engine-deploy.lock
flock -n 9 || {
  echo "⚠️  Another deployment is already running."
  echo "🛌 Taking a nap instead of causing chaos."
  exit 0
}

echo ""
echo "======================================================"
echo "🚀 Engine Deployment"
echo "🌿 Branch : $BRANCH"
echo "📂 Project: $APP_DIR"
echo "======================================================"
echo ""

echo "📡 [1/6] Contacting origin..."
git fetch origin

echo "📥 [2/6] Pulling the latest code..."
git reset --hard "origin/$BRANCH"

echo ""
echo "📦 [3/6] Installing dependencies..."
bun install --frozen-lockfile

echo ""
echo "🧬 [4/6] Generating ORM artifacts + migrations..."
(
  cd api
  bun run generate
  bun run migrate
)

echo ""
echo "🏗️  [5/6] Forging application builds..."
bun run build

echo ""
echo "♻️  [6/6] Respawning services..."

for SVC in $SERVICES; do
  echo "   🔄 Restarting $SVC..."
  sudo /usr/bin/supervisorctl restart "$SVC"
  echo "   ✅ $SVC is back online."
done

echo ""
echo "======================================================"
echo "🎉 QUEST COMPLETED!"
echo "✅ Deployment finished successfully."
echo "☕ Time to grab a coffee... until the next bug appears."
echo "======================================================"