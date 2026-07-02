#!/usr/bin/env bash
# Remove the demo resources. Pass --all to also stop/delete the minikube VM.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Deleting the application resources"
kubectl delete -k k8s/overlays/local --ignore-not-found=true || true

if [[ "${1:-}" == "--all" ]]; then
  echo "==> Deleting the minikube cluster"
  minikube delete
else
  echo "Namespace/app removed. To delete the whole minikube cluster: $0 --all"
fi
