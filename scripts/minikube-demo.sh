#!/usr/bin/env bash
#
# One-command local Kubernetes demo on minikube.
#
# It starts minikube, builds the app image INSIDE minikube's Docker daemon
# (so no registry/push is needed), deploys the local kustomize overlay, waits
# for a healthy rollout, and prints how to reach the app.
#
# Prerequisites: minikube, kubectl, docker.
#
# Usage:
#   ./scripts/minikube-demo.sh
#
set -euo pipefail

cd "$(dirname "$0")/.."

NS="devops-showcase"
IMAGE="devops-showcase:local"

step() { printf "\n\033[1;36m==> %s\033[0m\n" "$1"; }
need() {
  command -v "$1" >/dev/null 2>&1 && return 0
  echo "ERROR: '$1' is required but not installed."
  case "$1" in
    docker)   echo "  Install:  brew install --cask docker   (then launch it: open -a Docker)";;
    minikube) echo "  Install:  brew install minikube";;
    kubectl)  echo "  Install:  brew install kubectl";;
  esac
  echo "  See docs/MINIKUBE.md for the full prerequisites list."
  exit 1
}

need docker
need minikube
need kubectl

# Docker must actually be running (not just installed) for the docker driver.
if ! docker info >/dev/null 2>&1; then
  echo "ERROR: Docker is installed but the engine is not running."
  echo "  Start Docker Desktop:  open -a Docker"
  echo "  Wait until 'docker version' shows a Server section, then re-run this script."
  exit 1
fi

step "1/6 Starting minikube (docker driver)"
minikube status >/dev/null 2>&1 || minikube start --driver=docker --cpus=2 --memory=2200

step "2/6 Enabling the ingress (nginx) and metrics-server addons"
minikube addons enable ingress >/dev/null
minikube addons enable metrics-server >/dev/null || true

step "3/6 Building the image inside minikube's Docker daemon"
# Point docker at minikube so the image is available to the cluster directly.
eval "$(minikube -p minikube docker-env)"
docker build \
  --build-arg VERSION="minikube" \
  --build-arg COMMIT="$(git -C . rev-parse --short HEAD 2>/dev/null || echo local)" \
  --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -t "${IMAGE}" .
# Restore the normal docker context for the rest of the script.
eval "$(minikube -p minikube docker-env -u)"

step "4/6 Deploying the local overlay with kubectl"
kubectl apply -k k8s/overlays/local

step "5/6 Waiting for a healthy rollout"
kubectl -n "${NS}" rollout status deployment/devops-showcase --timeout=120s
kubectl -n "${NS}" get pods -o wide

step "6/6 Access"
IP="$(minikube ip)"
cat <<EOF

The app is deployed to namespace '${NS}' with 2 replicas.

Option A — via Ingress (nginx):
  1) Add this line to /etc/hosts (needs sudo):
       ${IP}  devops-showcase.local
  2) Open:  http://devops-showcase.local/
     Health: http://devops-showcase.local/healthz
     Metrics: http://devops-showcase.local/metrics

Option B — via port-forward (no /etc/hosts change):
     kubectl -n ${NS} port-forward svc/devops-showcase 8080:80
     # then: curl localhost:8080/healthz

Generate traffic (Option B running):
     for i in \$(seq 1 100); do curl -s localhost:8080/api/work >/dev/null; curl -s localhost:8080/api/error >/dev/null; done

Watch it self-heal (delete a pod, K8s recreates it):
     kubectl -n ${NS} delete pod -l app.kubernetes.io/name=devops-showcase --wait=false
     kubectl -n ${NS} get pods -w

Optional observability (Prometheus Operator + Grafana + the ServiceMonitor):
     see docs/MINIKUBE.md  -> "Add Prometheus & Grafana"

Tear everything down:
     ./scripts/minikube-teardown.sh
EOF
