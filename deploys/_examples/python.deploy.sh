#!/bin/bash
set -e

# ===========================================================
# 🚀 Python app (venv + pip)
# ===========================================================

APP_DIR="/var/www/my-python-app"
BRANCH="main"
SERVICES="my-python-app"      # Supervisor services
VENV="venv"                   # nama folder virtualenv relatif ke APP_DIR
# --------------------------------------

export PATH="/usr/local/bin:/usr/bin:$PATH"

cd "$APP_DIR"

exec 9>/tmp/$(basename "$APP_DIR")-deploy.lock
flock -n 9 || { echo "⚠️  Another deployment is already running. Skipping."; exit 0; }

echo ""
echo "======================================================"
echo "🚀 Python App Deployment: $APP_DIR ($BRANCH)"
echo "======================================================"
echo ""

echo "📡 [1/5] Contacting origin & pulling latest code..."
git fetch origin
git reset --hard "origin/$BRANCH"

echo ""
echo "📦 [2/5] Installing dependencies..."
[ -d "$VENV" ] || python3 -m venv "$VENV"
"$VENV/bin/pip" install --upgrade pip
"$VENV/bin/pip" install -r requirements.txt

# Kalau ada lock file (pip-tools / uv), pakai itu lebih reproducible:
# "$VENV/bin/pip" install -r requirements.txt --require-hashes

echo ""
echo "🧬 [3/5] Running migrations (kalau ada)..."
if [ -f "manage.py" ]; then
  "$VENV/bin/python" manage.py migrate --noinput
elif [ -f "alembic.ini" ]; then
  "$VENV/bin/alembic" upgrade head
else
  echo "   (no manage.py / alembic.ini, skipped)"
fi

echo ""
echo "♻️  [4/5] Respawning services..."
for SVC in $SERVICES; do
  echo "   🔄 Restarting $SVC..."
  sudo /usr/bin/supervisorctl restart "$SVC"
done

echo ""
echo "✅ [5/5] Deployment finished."
