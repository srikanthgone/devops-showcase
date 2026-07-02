#!/usr/bin/env bash
# Convenience wrapper to bring up the full local stack and print the URLs.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Building and starting the stack (app + Traefik + Prometheus + Grafana)"
docker compose up --build -d

cat <<'EOF'

Stack is starting. Give it ~10s, then open:

  App (via Traefik) : http://app.localhost
  Traefik dashboard : http://localhost:8090
  Prometheus        : http://localhost:9090
  Grafana           : http://localhost:3000   (admin / admin)

Generate traffic:   ./scripts/smoke-test.sh
Tear down:          docker compose down -v
EOF
