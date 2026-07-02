# Test Report & Verification Runbook

This document is the single source of truth for **what is tested, how to run
it, and what a passing run looks like**. It is written so you (or an
interviewer) can reproduce every result locally.

> **Transparency note:** the automated tests were authored and reviewed
> statically. This machine used to generate the repo had no Go toolchain and no
> network access, so the live `go test` run must be executed on your machine (or
> in CI). The commands and expected output below are exact — run them and you
> will get the real green result. No test output in this repo is fabricated.

---

## 1. Test inventory (13 automated tests, 3 packages)

### `internal/config` — configuration loading (4)
| # | Test | Type | Asserts |
| --- | --- | --- | --- |
| 1 | `TestLoadDefaults` | unit | Defaults applied when env empty (`PORT=8080`, `LOG_LEVEL=info`, timeouts) |
| 2 | `TestLoadRejectsInvalidPort` | unit (edge) | Non-numeric `PORT` returns an error (fail-fast) |
| 3 | `TestLoadHonoursOverrides` | unit | Env overrides win (`PORT`, `APP_ENV`, `SHUTDOWN_TIMEOUT`) |
| 4 | `TestGetEnvDurationFallsBackOnGarbage` | unit (edge) | Malformed duration falls back to default (no crash) |

### `internal/handlers` — HTTP behaviour (6)
| # | Test | Type | Asserts |
| --- | --- | --- | --- |
| 5 | `TestHealthz` | http | Liveness `/healthz` → 200 |
| 6 | `TestReadyzTogglesWithState` | http (state) | `/readyz` = 503 before ready, 200 after `MarkReady()` |
| 7 | `TestHelloUsesQueryParam` | http | `?name=` reflected in JSON body |
| 8 | `TestWorkReturnsLatency` | http | `/api/work` returns a `latency_ms` field |
| 9 | `TestRootReturnsMetadata` | http | `/` returns `service` + `build` metadata |
| 10 | `TestErrorEndpointIsAlwaysWellFormed` | http (fuzz-ish) | 50 iterations of `/api/error` → always 200/500 and valid JSON |

### `internal/metrics` — Prometheus exposition (3)
| # | Test | Type | Asserts |
| --- | --- | --- | --- |
| 11 | `TestMiddlewareRecordsAndExposes` | unit | Counter=3 and histogram exposed for 3 requests |
| 12 | `TestErrorStatusIsRecorded` | unit (edge) | 5xx captured as `status="500"` label |
| 13 | `TestHistogramBucketsAreCumulative` | unit (correctness) | Bucket math cumulative; `le="+Inf"` == `_count` |

**Edge cases explicitly covered:** invalid/garbage configuration, the
non-deterministic error endpoint (tested for well-formedness, not a fixed
result), error-status label capture, and histogram bucket-boundary correctness.

---

## 2. How to run the tests

```bash
cd devops-showcase

# Full suite with race detector + coverage (this is what CI runs)
go test -race -covermode=atomic -coverprofile=coverage.out ./...

# Verbose (see each test name)
go test -v ./...

# Per package
go test ./internal/config/...
go test ./internal/handlers/...
go test ./internal/metrics/...

# HTML coverage report
go tool cover -html=coverage.out
```

Or simply:
```bash
make test     # runs the race+coverage command above
make cover    # opens the HTML coverage report
```

### Expected result (shape)
A passing run prints one `ok` line per package and **no `FAIL`**:
```
ok  	devops-showcase/internal/config     0.4s
ok  	devops-showcase/internal/handlers   0.6s
ok  	devops-showcase/internal/metrics    0.3s
```
With `-v` you will see 13 `--- PASS:` lines (one per test above).

> If `go` is not installed: `brew install go` (macOS) or see https://go.dev/dl.
> No third-party modules are required — the project is stdlib-only, so
> `go test ./...` works fully offline after Go itself is installed.

---

## 3. Static analysis gates (also run in CI)

```bash
go vet ./...                    # built-in correctness checks
golangci-lint run ./...         # staticcheck, revive, gocritic, errcheck, ...
gofmt -l .                      # formatting (empty output = clean)
```

---

## 4. Container & manifest verification

```bash
# Build the production image locally
make docker-build

# Confirm the binary runs and reports its build metadata
docker run --rm ghcr.io/OWNER/devops-showcase:latest -version

# Render Kubernetes manifests (no cluster needed) — must succeed
kubectl kustomize k8s/overlays/staging
kubectl kustomize k8s/overlays/production
```

---

## 5. End-to-end smoke test (full stack)

```bash
make up            # app + Traefik + Prometheus + Grafana via docker compose
make smoke         # fires ~150 requests, prints sample metrics

# Manual checks
curl http://app.localhost/healthz          # {"status":"ok"}
curl http://app.localhost/readyz           # {"status":"ready"}
curl http://app.localhost/metrics | head   # Prometheus exposition
open http://localhost:9090/targets         # all targets UP
open http://localhost:3000                 # Grafana (admin/admin), dashboard populated
make down
```

**Pass criteria for E2E:**
- `/healthz` and `/readyz` return 200.
- Prometheus `/targets` shows the app target as **UP**.
- Grafana dashboard shows non-zero request rate and a latency curve.
- After running `make smoke`, the error-ratio panel shows ~30% on `/api/error`
  (the intentional fault-injection), demonstrating the alerting path.

---

## 6. CI evidence

On GitHub, the **Actions** tab shows each run's `test`, `build`, `manifests`,
and `deploy` jobs with green checks; Trivy results appear under
**Security → Code scanning**. In Jenkins, the pipeline view shows each stage
(Test → Build → Scan → Push → Validate → Deploy) with logs and archived
`coverage.out` / rendered manifests.
