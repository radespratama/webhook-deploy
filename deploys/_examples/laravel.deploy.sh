#!/bin/bash
set -e

# ===========================================================
# 🚀 Laravel / PHP (composer + artisan)
# ===========================================================

APP_DIR="/var/www/my-laravel-app"
BRANCH="main"
SERVICES=""                   
PHP_FPM="php8.3-fpm"          
# --------------------------------------

export PATH="/usr/local/bin:/usr/bin:$PATH"
COMPOSER="composer"

cd "$APP_DIR"

exec 9>/tmp/$(basename "$APP_DIR")-deploy.lock
flock -n 9 || { echo "⚠️  Another deployment is already running. Skipping."; exit 0; }

echo ""
echo "======================================================"
echo "🚀 Laravel Deployment: $APP_DIR ($BRANCH)"
echo "======================================================"
echo ""

echo "📡 [1/6] Contacting origin & pulling latest code..."
git fetch origin
git reset --hard "origin/$BRANCH"

echo ""
echo "📦 [2/6] Installing dependencies..."
$COMPOSER install --no-dev --optimize-autoloader --no-interaction

echo ""
echo "🧬 [3/6] Running migrations..."
php artisan migrate --force

echo ""
echo "🗂️  [4/6] Clearing & rebuilding caches..."
php artisan config:cache
php artisan route:cache
php artisan view:cache

echo ""
echo "🔗 [5/6] Linking storage (idempotent)..."
php artisan storage:link || true

echo ""
echo "♻️  [6/6] Respawning services..."
[ -n "$SERVICES" ] && for SVC in $SERVICES; do
  echo "   🔄 Restarting $SVC..."
  sudo /usr/bin/supervisorctl restart "$SVC"
done

echo ""
echo "✅ Deployment finished."
