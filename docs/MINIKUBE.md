# Running the app on Kubernetes with minikube

This is a **live, local Kubernetes deployment** you can run on your Mac and show
in an interview — the real thing (pods, Service, Ingress, rolling updates,
self-healing), not just docker-compose.

---

## Prerequisites

Install once (macOS with Homebrew):
```bash
brew install minikube kubectl
# Docker Desktop must be running (minikube uses it as the driver)
```

Check:
```bash
minikube version && kubectl version --client && docker version
```

---

## One command

```bash
make minikube-up          # or: ./scripts/minikube-demo.sh
```

What the script does (each step is printed as it runs):
1. **Starts minikube** (docker driver, 2 CPU / ~2.2 GB).
2. **Enables addons**: `ingress` (nginx controller) and `metrics-server`.
3. **Builds the image inside minikube's Docker daemon** so the cluster can use
   it directly — no registry, no push, no pull.
4. **Deploys** `k8s/overlays/local` with `kubectl apply -k`.
5. **Waits** for a healthy rollout and lists the pods.
6. **Prints access instructions.**

---

## Accessing the app

**Option A — Ingress (nginx):**
```bash
# The script prints your cluster IP; add it to /etc/hosts:
echo "$(minikube ip)  devops-showcase.local" | sudo tee -a /etc/hosts

curl http://devops-showcase.local/healthz     # {"status":"ok"}
curl http://devops-showcase.local/readyz       # {"status":"ready"}
curl http://devops-showcase.local/             # service + build metadata
open  http://devops-showcase.local/metrics     # Prometheus exposition
```

**Option B — port-forward (no hosts change):**
```bash
kubectl -n devops-showcase port-forward svc/devops-showcase 8080:80
curl localhost:8080/healthz
```

---

## Things to demonstrate live

**Show the running pods and the Service:**
```bash
kubectl -n devops-showcase get pods,svc,ingress -o wide
```

**Self-healing** — delete a pod and watch Kubernetes recreate it:
```bash
kubectl -n devops-showcase delete pod -l app.kubernetes.io/name=devops-showcase --wait=false
kubectl -n devops-showcase get pods -w
```

**Zero-downtime rolling update** — change something and re-deploy while curling:
```bash
# In one terminal, hammer the endpoint:
while true; do curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/healthz; sleep 0.2; done

# In another, trigger a rollout (e.g. bump an env value) and watch — no 5xx:
kubectl -n devops-showcase set env deploy/devops-showcase DEMO_BUMP="$(date +%s)"
kubectl -n devops-showcase rollout status deploy/devops-showcase
```

**Readiness gating** — describe a pod and show the liveness/readiness probes:
```bash
kubectl -n devops-showcase describe pod -l app.kubernetes.io/name=devops-showcase | sed -n '/Liveness/,/Events/p'
```

---

## Add Prometheus & Grafana (optional, full observability)

The local overlay omits the `ServiceMonitor` because vanilla minikube has no
Prometheus Operator CRDs. To get the full monitoring stack:

```bash
# Install kube-prometheus-stack (Prometheus Operator + Prometheus + Grafana)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kps prometheus-community/kube-prometheus-stack -n monitoring --create-namespace

# Now the ServiceMonitor CRD exists — apply our monitoring resources:
kubectl apply -f k8s/base/servicemonitor.yaml -n devops-showcase

# Open Grafana (default admin password shown by the command below):
kubectl -n monitoring get secret kps-grafana -o jsonpath='{.data.admin-password}' | base64 -d; echo
kubectl -n monitoring port-forward svc/kps-grafana 3000:80
# Grafana at http://localhost:3000  (user: admin)
```

Then import the dashboard from `deploy/grafana/dashboards/app-dashboard.json`,
or point Prometheus at the app and run the PromQL from `docs/SPEC.md`.

> Note: the `ServiceMonitor` in `k8s/base/servicemonitor.yaml` selects on the
> label `release: kube-prometheus-stack`. If you install the chart with a
> different release name (above it is `kps`), update that label or the
> Prometheus Operator's `serviceMonitorSelector` accordingly.

---

## Tear down

```bash
make minikube-down            # removes the app
make minikube-down ARGS=--all # also deletes the minikube cluster
```

---

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `ErrImageNeverPull` on pods | The image wasn't built into minikube's daemon. Re-run the script, or run `eval $(minikube docker-env)` then `docker build -t devops-showcase:local .` |
| Ingress host not reachable | Ensure `minikube addons enable ingress`, and that `/etc/hosts` maps `devops-showcase.local` to `minikube ip`. Or use port-forward. |
| Pods `Pending` | minikube needs more resources: `minikube start --cpus=2 --memory=2200`. |
| HPA shows `<unknown>` targets | `minikube addons enable metrics-server` and wait ~30s. |
