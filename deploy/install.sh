#!/bin/sh
# Install or update the family habit tracker on the Axon EC2 (Ubuntu). `make deploy` copies the
# linux binary and this folder into a temporary directory on the server and runs it there:
#   sudo sh install.sh
# Re-running it snapshots the database, replaces the binary, the unit and the nginx site, and
# restarts the service. Settings below are the single source of truth for the unit and the site.
set -e
cd "$(dirname "$0")"

ADDR=127.0.0.1:7778
TZ_NAME=Asia/Kuala_Lumpur
SITE=family-habit-tracker.axon.commun.cloud

BIN=/opt/habit-tracker/bin
DATA=/var/lib/habit-tracker
SNAPSHOTS=$DATA/pre-deploy
KEEP_SNAPSHOTS=10

render() { sed -e "s|@ADDR@|$ADDR|g" -e "s|@TZ@|$TZ_NAME|g" -e "s|@SITE@|$SITE|g" "$1"; }

[ -f ./habit-tracker ] || { echo "no ./habit-tracker binary beside install.sh (run make deploy)"; exit 1; }

id -u habit-tracker >/dev/null 2>&1 || useradd -r -s /usr/sbin/nologin -d "$DATA" habit-tracker
mkdir -p "$BIN" "$DATA" "$SNAPSHOTS"
chown -R habit-tracker:habit-tracker "$DATA"
chmod 750 "$DATA"

# A verified snapshot before touching a live database, taken with the binary that wrote it.
# These live apart from the app's own daily backups so they never push a daily out of rotation.
if [ -f "$DATA/habits.sqlite" ] && [ -x "$BIN/habit-tracker" ]; then
  runuser -u habit-tracker -- "$BIN/habit-tracker" -db "$DATA/habits.sqlite" \
    -backup "$SNAPSHOTS/pre-deploy-$(date -u +%Y%m%dT%H%M%SZ).sqlite"
  ls -1t "$SNAPSHOTS"/pre-deploy-*.sqlite | tail -n +$((KEEP_SNAPSHOTS + 1)) | xargs -r rm -f
fi

# Linux refuses to overwrite a running binary, but renaming over it is fine.
cp ./habit-tracker "$BIN/habit-tracker.new"
chmod 755 "$BIN/habit-tracker.new"
mv -f "$BIN/habit-tracker.new" "$BIN/habit-tracker"

render habit-tracker.service > /etc/systemd/system/habit-tracker.service
systemctl daemon-reload
systemctl enable habit-tracker >/dev/null 2>&1
systemctl restart habit-tracker

tries=0
until curl -fsS "http://$ADDR/healthz" >/dev/null 2>&1; do
  tries=$((tries + 1))
  if [ "$tries" -ge 20 ]; then
    echo "habit-tracker did not become healthy; nginx left unchanged"
    journalctl -u habit-tracker -n 30 --no-pager
    exit 1
  fi
  sleep 0.5
done

render nginx.conf > "/etc/nginx/sites-available/$SITE"
ln -sf "/etc/nginx/sites-available/$SITE" "/etc/nginx/sites-enabled/$SITE"
nginx -t
systemctl reload nginx

echo "habit-tracker is serving https://$SITE (journalctl -u habit-tracker -f for logs)"
