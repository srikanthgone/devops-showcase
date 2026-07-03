# Learning Path — from zero to explaining this whole project

You are a learner and this repo touches a lot of tools. This document is a
**guided curriculum**: learn each tool in the right order, using *this project*
as your lab. For every phase you get:

- **What / Why** — one-line mental model
- **Read in this repo** — the exact file(s) to open
- **Official docs** — where to learn it properly
- **Try it** — a small hands-on exercise you can run

> Golden rule: **run it first, then read the file, then read the docs.** Seeing
> a thing work makes the theory stick.

---

## The big picture (read this first)

The whole system is one sentence:

> A developer **pushes code** → CI **tests, builds a container, scans, and
> publishes** it → it is **deployed to Kubernetes** → **Traefik** routes user
> traffic to it → **Prometheus** collects metrics → **Grafana** shows
> dashboards → **alerts** fire if something is wrong.

```
Code → Git push → CI/CD → Container image → Kubernetes → Ingress → Users
                                                   ↑
                                   Prometheus scrapes metrics → Grafana + Alerts
```

Keep coming back to this picture — every tool below is one box in it.

---

## Phase 0 — Foundations (a few days)

You need a little of these before the rest makes sense. Don't over-study; get
comfortable, then move on.

| Topic | Why | Resource |
| --- | --- | --- |
| Terminal / Linux basics | Everything runs via CLI | [Missing Semester (MIT)](https://missing.csail.mit.edu/) |
| YAML | K8s, CI, configs are all YAML | [Learn YAML in Y minutes](https://learnxinyminutes.com/docs/yaml/) |
| HTTP basics | The app is an HTTP service | [MDN: HTTP overview](https://developer.mozilla.org/en-US/docs/Web/HTTP/Overview) |
| Git & GitHub | Source control + triggers CI | [Git handbook](https://docs.github.com/en/get-started/using-git) |

**Try it:** `git log --oneline` in this repo and read the commit messages.

---

## Phase 1 — The application (the thing we ship)

**What/Why:** A small Go web service. You don't need to master Go — just
understand what it exposes: some API endpoints, a health check, and metrics.

**Read in this repo:**
- `cmd/server/main.go` — startup, config, graceful shutdown
- `internal/handlers/handlers.go` — the endpoints (`/healthz`, `/api/work`, …)
- `internal/metrics/metrics.go` — how `/metrics` is produced

**Official docs:** [Go tour](https://go.dev/tour/) (optional), [net/http](https://pkg.go.dev/net/http)

**Try it (needs Go — `brew install go`):**
```bash
make run
# in another terminal:
curl localhost:8080/healthz
curl localhost:8080/api/hello?name=srikanth
curl localhost:8080/metrics
```

---

## Phase 2 — Docker (package the app into a container)

**What/Why:** A container bundles the app + everything it needs into one
portable image that runs identically everywhere. This is the unit CI builds and
Kubernetes runs.

**Read in this repo:**
- `Dockerfile` — note the **two stages** (build, then a tiny distroless runtime)
- `.dockerignore` — what is kept out of the build

**Official docs:** [Docker get started](https://docs.docker.com/get-started/), [multi-stage builds](https://docs.docker.com/build/building/multi-stage/), [distroless](https://github.com/GoogleContainerTools/distroless)

**Try it (needs Docker running):**
```bash
make docker-build
docker run --rm -p 8080:8080 ghcr.io/OWNER/devops-showcase:latest
curl localhost:8080/healthz
docker images | grep devops-showcase   # see how small the image is
```
**Concept to internalise:** *why distroless?* (no shell/OS packages → fewer
vulnerabilities). This is a favourite interview question.

---

## Phase 3 — Kubernetes core (run containers at scale)

**What/Why:** Kubernetes (K8s) runs your containers, keeps the right number
alive, restarts crashed ones, and load-balances traffic. The three core objects:
- **Pod** = one running instance of your container
- **Deployment** = "keep N identical pods running, roll out new versions safely"
- **Service** = a stable internal address that load-balances across pods

**Read in this repo:**
- `k8s/base/deployment.yaml` — replicas, probes, resources, security
- `k8s/base/service.yaml` — the stable address
- `k8s/base/namespace.yaml` — isolation + security policy

**Official docs:** [K8s basics tutorial](https://kubernetes.io/docs/tutorials/kubernetes-basics/), [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), [Services](https://kubernetes.io/docs/concepts/services-networking/service/)

**Try it (minikube):** follow `docs/MINIKUBE.md`, then:
```bash
kubectl -n devops-showcase get pods            # see your pods
kubectl -n devops-showcase get deploy,svc      # deployment + service
kubectl -n devops-showcase delete pod -l app.kubernetes.io/name=devops-showcase
kubectl -n devops-showcase get pods -w         # watch K8s recreate it (self-healing!)
```

---

## Phase 4 — Kubernetes production features

**What/Why:** The difference between "it runs" and "it runs reliably".

| Feature | File | What it gives you |
| --- | --- | --- |
| Liveness/Readiness/Startup probes | `k8s/base/deployment.yaml` | K8s restarts unhealthy pods; only sends traffic to ready ones |
| Resource requests/limits | same | Fair scheduling, no noisy-neighbour |
| Security context | same | Non-root, read-only FS, dropped privileges |
| HorizontalPodAutoscaler | `k8s/base/hpa.yaml` | Auto add/remove pods under load |
| PodDisruptionBudget | `k8s/base/pdb.yaml` | Keep minimum pods during maintenance |
| **Kustomize** overlays | `k8s/overlays/` | One base, different settings per env (staging/prod/local) |

**Official docs:** [Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), [HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/), [Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)

**Try it:**
```bash
kubectl kustomize k8s/overlays/production   # see the fully-rendered manifests
diff <(kubectl kustomize k8s/overlays/staging) <(kubectl kustomize k8s/overlays/production)
```

---

## Phase 5 — Ingress & Traefik (let users in)

**What/Why:** A Service is only reachable *inside* the cluster. An **Ingress**
(implemented by **Traefik**) is the front door that routes outside traffic to
the right Service based on hostname/path.

**Read in this repo:**
- `k8s/base/ingressroute.yaml` — Traefik's own CRD
- `k8s/base/ingress.yaml` — the portable, standard Ingress
- `docker-compose.yml` — see Traefik configured for the local stack

**Official docs:** [K8s Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/), [Traefik docs](https://doc.traefik.io/traefik/)

**Try it:** `docker compose up` then open http://localhost:8090 (Traefik
dashboard) and http://app.localhost.

---

## Phase 6 — Observability: Prometheus + Grafana

**What/Why:** You can't operate what you can't see.
- **Prometheus** pulls (`scrapes`) numeric **metrics** from `/metrics` and stores
  them; it also evaluates **alert rules**.
- **Grafana** draws dashboards from Prometheus data.
- The mental model for app metrics is **RED**: **R**ate, **E**rrors, **D**uration.

**Read in this repo:**
- `deploy/prometheus/prometheus.yml` — what gets scraped
- `deploy/prometheus/alerts.yml` — when to alert (down / errors / slow)
- `deploy/grafana/` — datasource + dashboard, provisioned as code
- `internal/metrics/metrics.go` — where the numbers come from

**Official docs:** [Prometheus](https://prometheus.io/docs/introduction/overview/), [PromQL basics](https://prometheus.io/docs/prometheus/latest/querying/basics/), [Grafana](https://grafana.com/docs/grafana/latest/getting-started/)

**Try it:** `docker compose up`, then `./scripts/smoke-test.sh` to make traffic,
then open Grafana at http://localhost:3000 (admin/admin) and Prometheus at
http://localhost:9090/targets. Run this in Prometheus:
```promql
sum by (path) (rate(http_requests_total[1m]))
```

---

## Phase 7 — CI/CD: GitHub Actions & Jenkins

**What/Why:** Automate everything from "push" to "deployed". Same pipeline
stages, two tools, so you can talk about either.

**Read in this repo:**
- `.github/workflows/ci-cd.yaml` — GitHub Actions (runs on every push)
- `Jenkinsfile` — the same stages in Jenkins syntax

**Official docs:** [GitHub Actions](https://docs.github.com/en/actions/learn-github-actions), [Jenkins pipelines](https://www.jenkins.io/doc/book/pipeline/)

**Try it:** Go to your repo's **Actions** tab and watch the workflow run after a
push. Open a job and read each step's log.

---

## Phase 8 — Security & supply chain

**What/Why:** Ship securely. **Trivy** scans images for known vulnerabilities;
distroless + non-root shrink the attack surface; images are pinned by digest.

**Read in this repo:** the `Trivy` steps in `.github/workflows/ci-cd.yaml` and
`Jenkinsfile`; the `securityContext` in `k8s/base/deployment.yaml`;
`docs/PRODUCTION-READINESS.md` (Security section).

**Official docs:** [Trivy](https://aquasecurity.github.io/trivy/), [K8s Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)

---

## Suggested schedule (part-time)

| Week | Focus |
| --- | --- |
| 1 | Phase 0–1: foundations + run the app locally |
| 2 | Phase 2: Docker (build, run, understand the image) |
| 3 | Phase 3: Kubernetes core on minikube |
| 4 | Phase 4: probes, HPA, kustomize |
| 5 | Phase 5–6: ingress + Prometheus/Grafana |
| 6 | Phase 7–8: CI/CD + security, then rehearse the `docs/SPEC.md` walkthrough |

## Great free resources (curated)
- [Kubernetes official tutorials](https://kubernetes.io/docs/tutorials/)
- [play-with-docker.com](https://labs.play-with-docker.com/) / [play-with-k8s.com](https://labs.play-with-k8s.com/) — browser sandboxes
- [KodeKloud](https://kodekloud.com/) — hands-on labs (Docker/K8s/CI)
- [Prometheus + Grafana getting started](https://grafana.com/docs/grafana/latest/getting-started/get-started-grafana-prometheus/)
- The [CNCF landscape](https://landscape.cncf.io/) — see how these tools fit the ecosystem

## How to prove you learned it
When you can do the `docs/SPEC.md` **10-minute walkthrough** from memory —
explaining *what each tool is, why it's there, and what it achieves* — you're
ready to present this in an interview.
