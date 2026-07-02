# devops-showcase Makefile
# Run `make help` to list targets.

APP        := devops-showcase
PKG        := ./...
IMAGE      ?= ghcr.io/OWNER/devops-showcase
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
  -X $(APP)/internal/version.Version=$(VERSION) \
  -X $(APP)/internal/version.Commit=$(COMMIT) \
  -X $(APP)/internal/version.BuildDate=$(BUILD_DATE)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the server locally
	go run ./cmd/server

.PHONY: build
build: ## Build the binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: test
test: ## Run unit tests with race detector + coverage
	go test -race -covermode=atomic -coverprofile=coverage.out $(PKG)

.PHONY: cover
cover: test ## Open the HTML coverage report
	go tool cover -html=coverage.out

.PHONY: vet
vet: ## Run go vet
	go vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed)
	golangci-lint run ./...

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: docker-build
docker-build: ## Build the container image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

.PHONY: up
up: ## Start the full local stack (app + Traefik + Prometheus + Grafana)
	docker compose up --build

.PHONY: down
down: ## Tear down the local stack
	docker compose down -v

.PHONY: smoke
smoke: ## Run the smoke test against the running stack
	./scripts/smoke-test.sh

.PHONY: k8s-render
k8s-render: ## Render kustomize overlays
	kubectl kustomize k8s/overlays/staging
	kubectl kustomize k8s/overlays/production

.PHONY: k8s-deploy
k8s-deploy: ## Apply the production overlay to the current kube-context
	kubectl apply -k k8s/overlays/production

.PHONY: minikube-up
minikube-up: ## One-command local Kubernetes demo on minikube
	./scripts/minikube-demo.sh

.PHONY: minikube-down
minikube-down: ## Remove the minikube demo (add ARGS=--all to delete the cluster)
	./scripts/minikube-teardown.sh $(ARGS)
