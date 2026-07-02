#!/usr/bin/env bash
# Smoke test the running stack: hit the app through Traefik and generate some
# traffic so the Grafana dashboards light up.
set -euo pipefail

HOST="${HOST:-http://app.localhost}"
ITER="${ITER:-50}"

echo "==> Health checks"
curl -fsS "${HOST}/healthz" && echo
curl -fsS "${HOST}/readyz" && echo

echo "==> Root"
curl -fsS "${HOST}/" && echo

echo "==> Generating ${ITER} requests of traffic..."
for i in $(seq 1 "${ITER}"); do
  curl -fsS "${HOST}/api/hello?name=user${i}" >/dev/null || true
  curl -fsS "${HOST}/api/work" >/dev/null || true
  # This endpoint fails ~30% of the time on purpose.
  curl -fsS "${HOST}/api/error" >/dev/null || true
done

echo "==> Sample of exposed metrics"
curl -fsS "${HOST}/metrics" | grep -E '^http_requests_total' | head -n 10

echo
echo "Done. Open Grafana at http://localhost:3000 (admin/admin)."
