# Windows 10 — Complete Setup Guide

This guide takes a **fresh Windows 10 machine** to a fully working
devops-showcase environment: run the app, build the Docker image, bring up the
full stack (app + Traefik + Prometheus + Grafana), and deploy to a local
Kubernetes cluster with minikube.

There are two paths. **Choose one:**

- **Path A — WSL2 (Ubuntu) — RECOMMENDED.** This project uses bash scripts,
  `make`, and Linux tooling. Inside WSL2 everything "just works" exactly like
  the macOS/Linux instructions. **Pick this unless you have a reason not to.**
- **Path B — Native Windows (PowerShell).** Works too, with a few Windows
  quirks (no `make` by default, `.sh` scripts need Git Bash).

---

## 0. System requirements & one-time BIOS check

- **Windows 10 version 2004 (build 19041) or newer.** Check with `winver`.
- **64-bit CPU with virtualization enabled.** Open **Task Manager → Performance
  → CPU** and confirm **"Virtualization: Enabled"**. If it says Disabled, enable
  **Intel VT-x / AMD-V** (and, for WSL2, the virtualization features) in your
  BIOS/UEFI.
- At least **8 GB RAM** (16 GB recommended for the full stack + Kubernetes).

---

# PATH A — WSL2 (recommended)

## A1. Install WSL2 + Ubuntu

Open **PowerShell as Administrator** and run:

```powershell
wsl --install -d Ubuntu
```

Reboot if prompted. After reboot, Ubuntu opens and asks you to create a **UNIX
username and password** (remember this password — it's your `sudo` password).

Verify you are on WSL **2**:
```powershell
wsl --list --verbose
```
The `VERSION` column should say `2`. If it says `1`, run:
```powershell
wsl --set-version Ubuntu 2
wsl --set-default-version 2
```

## A2. Install Docker Desktop (with the WSL2 backend)

1. Download **Docker Desktop for Windows**: <https://www.docker.com/products/docker-desktop/>
2. During install, keep **"Use WSL 2 instead of Hyper-V"** checked.
3. After install, open Docker Desktop → **Settings → Resources → WSL Integration**
   → enable integration for your **Ubuntu** distro → **Apply & Restart**.
4. Leave Docker Desktop running (whale icon in the system tray).

Now **everything below runs inside the Ubuntu (WSL) terminal**, not PowerShell.
Open it from the Start menu ("Ubuntu").

## A3. Install the tools inside Ubuntu (WSL)

```bash
# Update package lists
sudo apt update

# Git, make, curl (usually present, safe to run)
sudo apt install -y git make curl

# Confirm Docker is reachable from WSL (Docker Desktop must be running)
docker version    # must show a "Server:" section

# Go (1.23+)
curl -fsSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz -o /tmp/go.tgz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
go version

# kubectl
curl -fsSLO "https://dl.k8s.io/release/$(curl -fsSL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && rm kubectl
kubectl version --client

# minikube
curl -fsSLO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube && rm minikube-linux-amd64
minikube version
```

## A4. Get the code

```bash
cd ~
git clone https://github.com/srikanthgone/devops-showcase.git
cd devops-showcase
```

## A5. Run things

```bash
# 1) Unit tests (13 tests, 3 packages)
make test

# 2) Run the app directly
make run
#   in another Ubuntu tab:
curl localhost:8080/healthz          # {"status":"ok"}

# 3) Build the container image
make docker-build

# 4) Full local stack: app + Traefik + Prometheus + Grafana
make up                               # docker compose up --build
#   then in another tab, generate traffic:
make smoke
```

Open these in your **Windows browser** (WSL forwards localhost automatically):
- Grafana: <http://localhost:3000> (admin / admin)
- Prometheus: <http://localhost:9090>
- Traefik dashboard: <http://localhost:8090>
- App via Traefik: <http://app.localhost>

Tear down: `make down`.

## A6. Deploy to Kubernetes (minikube)

```bash
make minikube-up
```
When it finishes, use port-forward (simplest) in a second tab:
```bash
kubectl -n devops-showcase port-forward svc/devops-showcase 8080:80
```
Then open <http://localhost:8080/healthz> and <http://localhost:8080/metrics>.

Explore the cluster:
```bash
kubectl -n devops-showcase get pods,svc,hpa
kubectl -n devops-showcase delete pod -l app.kubernetes.io/name=devops-showcase
kubectl -n devops-showcase get pods -w    # watch self-healing; Ctrl+C to stop
```
Tear down: `make minikube-down` (add `ARGS=--all` to delete the cluster).

**You're done with Path A.**

---

# PATH B — Native Windows (PowerShell)

Use this only if you don't want WSL2. Some helper scripts (`*.sh`) and `make`
targets won't run natively; equivalent raw commands are given.

## B1. Install tools with winget

Open **PowerShell as Administrator**:

```powershell
winget install --id Git.Git -e
winget install --id GoLang.Go -e
winget install --id Kubernetes.kubectl -e
winget install --id Kubernetes.minikube -e
winget install --id Docker.DockerDesktop -e
```

Close and reopen PowerShell so the new `PATH` takes effect. Start **Docker
Desktop** and wait for the whale icon. Verify:
```powershell
git --version
go version
kubectl version --client
minikube version
docker version   # needs a "Server:" section (Docker Desktop running)
```

> `make` is not installed by default on Windows. Either install it
> (`winget install ezwinports.make` or use Chocolatey `choco install make`) or
> just run the underlying commands shown below.

## B2. Get the code

```powershell
cd $HOME
git clone https://github.com/srikanthgone/devops-showcase.git
cd devops-showcase
```

## B3. Run things (raw commands, no make)

```powershell
# Unit tests
go test -race ./...

# Run the app
go run ./cmd/server
#   in another PowerShell tab:
curl.exe http://localhost:8080/healthz

# Build the image
docker build -t devops-showcase:local .

# Full local stack (Traefik + Prometheus + Grafana + app)
docker compose up --build
```
Open in browser: Grafana <http://localhost:3000> (admin/admin), Prometheus
<http://localhost:9090>, Traefik <http://localhost:8090>, app
<http://app.localhost>.

Generate traffic (PowerShell):
```powershell
1..100 | ForEach-Object { curl.exe -s http://app.localhost/api/work > $null; curl.exe -s http://app.localhost/api/error > $null }
```

## B4. minikube on native Windows

The `scripts/minikube-demo.sh` is a bash script; on native Windows run the steps
manually in PowerShell:

```powershell
minikube start --driver=docker
minikube addons enable ingress
minikube addons enable metrics-server

# Build the image straight into minikube (no registry needed)
minikube image build -t devops-showcase:local .

# Deploy the local overlay
kubectl apply -k k8s/overlays/local
kubectl -n devops-showcase rollout status deployment/devops-showcase --timeout=120s

# Access via port-forward
kubectl -n devops-showcase port-forward svc/devops-showcase 8080:80
```
Then open <http://localhost:8080/healthz>.

> If you prefer the `.sh` scripts and `make`, run them from **Git Bash**
> (installed with Git) rather than PowerShell — but Path A (WSL2) is the
> cleaner way to use those.

---

## Editing the hosts file (only if you want the `*.local` URLs)

To use `http://devops-showcase.local` instead of port-forward, edit the Windows
hosts file **as Administrator**:

- File: `C:\Windows\System32\drivers\etc\hosts`
- Add a line (replace with your `minikube ip` output):
  ```
  192.168.49.2  devops-showcase.local
  ```
- For the docker-compose `app.localhost` URL, no edit is needed — Windows
  resolves `*.localhost` to 127.0.0.1 automatically in most browsers.

> Using `kubectl port-forward` + `http://localhost:8080` avoids all hosts-file
> editing and is the recommended approach for demos.

---

## Windows-specific troubleshooting

| Symptom | Cause / Fix |
| --- | --- |
| `wsl --install` fails | Update Windows (Settings → Update). Ensure virtualization is enabled in BIOS. |
| `docker version` shows only "Client" | Docker Desktop isn't running, or WSL integration for Ubuntu is off (Settings → Resources → WSL Integration). |
| minikube: `Exiting due to DRV_...` | Docker Desktop must be running; start it, then `minikube start --driver=docker`. |
| `ErrImageNeverPull` on pods | The image isn't in minikube. WSL: re-run `make minikube-up`. Native: `minikube image build -t devops-showcase:local .`. |
| `make: command not found` | Native Windows has no make. Use the raw commands (Path B) or install make, or use WSL2. |
| `.sh` script errors / `\r` issues | You ran a bash script from PowerShell, or CRLF line endings. Run scripts from **WSL2** or **Git Bash**. |
| Port 8080 "address already in use" | Use another port: `kubectl -n devops-showcase port-forward svc/devops-showcase 8081:80` and open `:8081`. |
| Browser can't reach `devops-showcase.local` | Use port-forward + `localhost`, or add the hosts entry (see above). |
| Ubuntu can't see Docker | Start Docker Desktop on Windows first; WSL uses its engine. |

---

## Which files do what (quick map)

| You want to… | Command | Needs |
| --- | --- | --- |
| Run tests | `make test` / `go test ./...` | Go |
| Run the app | `make run` / `go run ./cmd/server` | Go |
| Build image | `make docker-build` / `docker build ...` | Docker |
| Full stack + Grafana | `make up` / `docker compose up --build` | Docker |
| Local Kubernetes | `make minikube-up` (WSL) / manual (native) | Docker + minikube + kubectl |

For the concepts behind each tool, read [`LEARNING-PATH.md`](LEARNING-PATH.md);
for the interview walkthrough, [`SPEC.md`](SPEC.md).
