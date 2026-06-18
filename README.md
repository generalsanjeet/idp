# Internal Developer Platform (IDP)

A production-grade Internal Developer Platform built from scratch in Go.
Developers register services, deploy them to Kubernetes, and monitor logs
and metrics — all through a single API or browser UI.

---

## Architecture

Developer (UI / curl)
│
▼
┌───────────────────────────────┐
│       IDP Control Plane       │
│         (Go + chi)            │
│                               │
│  /services  →  Postgres       │
│  /deploy    →  Git (go-git)   │
│  /logs      →  Loki           │
│  /metrics   →  Prometheus     │
└───────────────┬───────────────┘
│ git push
▼
GitHub (gitops-repo)
│
│ watches
▼
ArgoCD
│ deploys
▼
Kubernetes (Kind)
┌───────────────┐
│  payments     │
│  inventory    │
│  billing      │
│  hello-service│
└───────────────┘
│
┌───────┴───────┐
▼               ▼
Loki          Prometheus
(logs)          (metrics)

---

## What it does

- **Service Registry** — register a service with name, repo URL, and owner
- **GitOps Deploy** — deploying updates `values.yaml` in a GitOps repo,
  ArgoCD detects the change and deploys to Kubernetes automatically
- **Auto-bootstrap** — registering a new service automatically creates
  its Helm chart structure and ArgoCD Application manifest
- **Log streaming** — fetch real pod logs via Loki LogQL
- **Metrics** — fetch deployment health and replica count via Prometheus
- **React UI** — browser interface for all of the above
- **CI Pipeline** — GitHub Actions runs tests, builds Docker image,
  pushes to ghcr.io, and calls the IDP deploy endpoint automatically

---

## Tech stack

| Tool | Purpose | Why |
|------|---------|-----|
| Go | Control plane backend | Type-safe, fast, excellent k8s ecosystem |
| chi | HTTP router | Lightweight, idiomatic, middleware support |
| slog | Structured logging | Standard library, JSON output, leveled |
| Postgres | Service registry | Reliable, UNIQUE constraints, RETURNING clause |
| golang-migrate | DB migrations | Versioned SQL files, tracked, reversible |
| client-go | Kubernetes API | Official Go k8s client |
| go-git | Git operations | Pure Go, no git binary dependency |
| Helm | Service packaging | Templated k8s manifests per service |
| ArgoCD | GitOps engine | Watches Git, syncs cluster, self-heals |
| Loki + Promtail | Log aggregation | Label-based, integrates with k8s pods |
| Prometheus | Metrics | kube-state-metrics for deployment health |
| Kind | Local Kubernetes | Zero cloud cost, full k8s API |
| GitHub Actions | CI pipeline | Test → build → push → deploy on every commit |
| ghcr.io | Container registry | Free, integrated with GitHub Actions |

---

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| POST | /services | Register a new service |
| GET | /services | List all services |
| POST | /deploy/{service} | Deploy a service with a new image |
| GET | /logs/{service} | Fetch recent logs from Loki |
| GET | /metrics/{service} | Fetch metrics from Prometheus |

---

## Run locally

### Prerequisites

- Go 1.26+
- Docker
- Kind
- kubectl
- Helm
- ArgoCD CLI

### 1. Start infrastructure

```bash
# Postgres
docker run -d --name idp-postgres \
  -e POSTGRES_USER=idp \
  -e POSTGRES_PASSWORD=idp \
  -e POSTGRES_DB=idp \
  -p 5432:5432 postgres:16

# Kind cluster
kind create cluster --name idp

# Loki + Promtail
helm repo add grafana https://grafana.github.io/helm-charts
helm upgrade --install loki grafana/loki-stack \
  --namespace monitoring --create-namespace \
  --set promtail.enabled=true

# Prometheus
helm repo add prometheus-community \
  https://prometheus-community.github.io/helm-charts
helm upgrade --install prometheus \
  prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set grafana.enabled=false \
  --set alertmanager.enabled=false

# ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd \
  -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

### 2. Start port-forwards

```bash
make port-forwards
```

### 3. Configure environment

```bash
export DATABASE_URL="postgres://idp:idp@localhost:5432/idp?sslmode=disable"
export GITOPS_REPO_URL="https://github.com/yourname/gitops-repo"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
export MIGRATIONS_PATH="./migrations"
```

### 4. Run the IDP

```bash
make run
```

### 5. Open the UI

---

http://localhost:8080

## Project structure

idp/
├── cmd/server/main.go          # Entry point, wiring
├── internal/
│   ├── config/                 # 12-factor config loading
│   ├── db/                     # Postgres connection + migrations
│   ├── service/                # Service registry (model, store, handler)
│   ├── deploy/                 # GitOps deploy via go-git
│   ├── logs/                   # Loki log proxy
│   ├── metrics/                # Prometheus metrics proxy
│   └── health/                 # Health check
├── migrations/                 # Versioned SQL migration files
└── ui/                         # React single-page app

---

## Known limitations / future work

- ArgoCD Application manifests must be manually applied after
  bootstrapping (`kubectl apply -f gitops-repo/argocd/<service>.yaml`)
- Port-forwards must be restarted after machine reboot
- No authentication on the IDP API
- Single namespace — no team isolation yet
- No rollback endpoint (can be done via git revert)

---

## What I learned building this

This project was built following a strict "make it work → make it right →
make it better" methodology — no frameworks, no shortcuts. Every
architectural decision (sentinel errors, interface-based testing,
GitOps over direct k8s API calls, structured logging) was made
deliberately after understanding the problem it solves.



# few commands

argocd app list
argocd app get <application-naem>
ngrok http 8080

# Apply the ArgoCD application for test-service
kubectl apply -f /tmp/gitops-repo/argocd/test-service.yaml

# Wait a few seconds then sync
argocd app sync test-service

curl http://localhost:8080/services

curl -X POST http://localhost:8080/services \
  -H "Content-Type: application/json" \
  -d '{"name":"gateway","repo_url":"https://github.com/org/gateway","owner":"team-core"}'

curl -X POST http://localhost:8080/deploy/gateway \
  -H "Content-Type: application/json" \
  -d '{"image":"nginx:latest"}'

curl http://localhost:8080/logs/gateway
curl http://localhost:8080/metrics/gateway
 
# to  test promethus and loki in k8s
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9091:9090
curl http://localhost:9091/api/v1/query?query=up

kubectl port-forward -n monitoring svc/loki 3100:3100
curl http://localhost:3100/loki/api/v1/labels

kubectl port-forward -n argocd svc/argocd-server 8888:443
kubectl get secret -n argocd argocd-initial-admin-secret -o jsonpath="{.data.password}" | base 64 -d
argocd login localhost:8888 --username admin --insecure
argocd account get-user-info
