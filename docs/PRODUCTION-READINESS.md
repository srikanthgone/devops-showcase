# Production Readiness Checklist

This is an honest, auditable assessment. Each item is marked:

- ✅ **Implemented** in this repository
- 🔶 **Partial / demonstrated as a stub** (shape is shown; wire to your infra)
- ⬜ **Recommended next step** (deliberately out of scope; name-drop as maturity)

> The value of this document in an interview is showing you *think in
> checklists* and know the difference between "works on my laptop" and
> "production-ready".

---

## 1. Reliability & availability

| Item | Status | Where |
| --- | --- | --- |
| Liveness probe | ✅ | `k8s/base/deployment.yaml` `/healthz` |
| Readiness probe (traffic gating) | ✅ | `/readyz`, `internal/handlers` |
| Startup probe (slow-start safety) | ✅ | deployment `startupProbe` |
| Graceful shutdown, drain in-flight | ✅ | `cmd/server/main.go` `srv.Shutdown` |
| Readiness fails *before* drain on SIGTERM | ✅ | `MarkUnready()` then `Shutdown` |
| Zero-downtime rolling update | ✅ | `maxUnavailable: 0, maxSurge: 1` |
| PodDisruptionBudget | ✅ | `k8s/base/pdb.yaml` (`minAvailable: 2`) |
| Multiple replicas + anti-affinity/spread | ✅ | `replicas: 3`, topology spread |
| HTTP server timeouts (Slowloris guard) | ✅ | `internal/config` read/write/idle |
| Retry/circuit-breaking to dependencies | ⬜ | no external deps in this demo |

## 2. Scalability & performance

| Item | Status | Where |
| --- | --- | --- |
| Stateless application | ✅ | no local state; safe to scale horizontally |
| Horizontal Pod Autoscaler | ✅ | `k8s/base/hpa.yaml` (CPU 70% / mem 80%) |
| Resource requests & limits | ✅ | deployment `resources` |
| Latency histogram enables SLO-based scaling | ✅ | metric present; custom-metrics adapter = next step |
| Load/soak testing | 🔶 | `scripts/smoke-test.sh` generates traffic; k6/vegeta = next |

## 3. Security & supply chain

| Item | Status | Where |
| --- | --- | --- |
| Non-root container | ✅ | distroless `nonroot`, UID 65532 |
| Minimal base image (no shell/pkg mgr) | ✅ | `gcr.io/distroless/static` |
| Read-only root filesystem | ✅ | `securityContext.readOnlyRootFilesystem` |
| Drop ALL Linux capabilities | ✅ | `capabilities.drop: [ALL]` |
| seccomp RuntimeDefault | ✅ | pod `securityContext` |
| Restricted Pod Security Standard | ✅ | namespace labels |
| No privilege escalation | ✅ | `allowPrivilegeEscalation: false` |
| Image vulnerability scanning | ✅ | Trivy in both pipelines |
| Immutable, digest-pinned images | ✅ | SHA tags; deploy by digest |
| No secrets in source/repo | ✅ | config via env/ConfigMap only |
| Image signing (cosign) | ⬜ | recommended next |
| SBOM generation | ⬜ | recommended next (e.g. `syft`) |
| NetworkPolicies | ⬜ | recommended next |
| Secrets manager (Vault/External Secrets) | ⬜ | recommended next |

## 4. Observability

| Item | Status | Where |
| --- | --- | --- |
| RED metrics (rate/errors/duration) | ✅ | `internal/metrics` |
| Prometheus scrape config | ✅ | `deploy/prometheus/` + `ServiceMonitor` |
| Alert rules (down/error-rate/latency) | ✅ | `deploy/prometheus/alerts.yml` |
| Grafana dashboard as code | ✅ | `deploy/grafana/` provisioning |
| Structured JSON logs | ✅ | `log/slog` |
| Build metadata surfaced | ✅ | `/`, `/metrics`, `--version` |
| Distributed tracing (OpenTelemetry) | ⬜ | recommended next |
| Centralised logs (Loki/ELK) | ⬜ | recommended next |

## 5. Delivery & operability (CI/CD)

| Item | Status | Where |
| --- | --- | --- |
| Lint + vet gate | ✅ | both pipelines |
| Unit tests with race detector + coverage | ✅ | `go test -race -cover` |
| Manifest validation in CI | ✅ | `kubectl kustomize` overlays |
| Multi-arch image build | ✅ | Buildx amd64+arm64 |
| Deploy gated to `main` / environment | ✅ | GH `environment: production` |
| Rollout status verification | ✅ | Jenkins `rollout status` |
| Automated rollback | 🔶 | K8s auto-rolls back failed rollouts; explicit strategy = next |
| Progressive delivery (canary/blue-green) | ⬜ | Argo Rollouts/Flagger = next |
| GitOps (Argo CD/Flux) | ⬜ | deploy step is a hand-off point |

## 6. Engineering quality

| Item | Status | Where |
| --- | --- | --- |
| 12-factor configuration | ✅ | `internal/config` |
| Dependency-free build (reproducible) | ✅ | stdlib only; tiny `go.mod` |
| Unit tests + edge cases | ✅ | 13 tests, 3 packages |
| README + spec + runbook docs | ✅ | `README.md`, `docs/` |
| Makefile for common tasks | ✅ | `make help` |
| Linter config committed | ✅ | `.golangci.yml` |
| One-command local environment | ✅ | `docker compose up` |

---

## Verdict

For its intended scope — a **reference / interview showcase** — this repository
is **production-grade in structure and practice**: it is secure by default,
observable, resilient to rollouts and disruptions, autoscaling, and gated by a
real CI/CD pipeline.

To run it as a *real* revenue-serving service you would add the ⬜ items above
(signing/SBOM, tracing, centralised logging, secrets manager, network policies,
progressive delivery). Being able to state exactly what is done vs. what remains
is itself a mark of production maturity.
