<div align="center">

# RegPilot

**Lightweight Docker Registry management UI**

A clean, production-ready web interface for [Docker Distribution Registry](https://docs.docker.com/registry/) (HTTP API V2).  
Built for teams that want Harbor-level clarity without Harbor-level complexity.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

[Features](#-features) · [Quick Start](#-quick-start) · [Architecture](#-architecture) · [Configuration](#-configuration) · [Garbage Collection](#-garbage-collection) · [API](#-api-documentation) · [Security](#-security)

</div>

---

## Why RegPilot?

Most registry UIs are either too heavy (Harbor, Nexus) or too minimal (raw `curl` against `/v2/`).

**RegPilot** sits in the middle:

| Need | RegPilot |
|------|----------|
| Browse repos / tags / manifests | ✅ |
| Delete images safely (by digest) | ✅ |
| Multi-registry connections | ✅ |
| Auth + roles + audit log | ✅ |
| One `docker compose up` | ✅ |
| No Postgres / Redis / K8s required | ✅ |
| Fake storage metrics | ❌ Never |

The **registry remains the source of truth**. SQLite only stores app users, registry connection config, audit logs, and retention settings.

---

## ✨ Features

### Core
- **JWT authentication** with bcrypt password hashing
- **Roles:** `ADMIN` (full) · `VIEWER` (read-only)
- **Multiple registry connections** (HTTP/HTTPS + optional basic auth)
- **Encrypted registry credentials** (AES) — never returned in API responses
- Per-registry **self-signed TLS** option (never disables TLS globally)

### Registry operations
- Repository catalog with **search / sort / pagination**
- Tag listing with digest, created time, and size (when available)
- **Manifest inspection** — platform, layers, digests, total size
- **Image delete** by resolving tag → digest → `DELETE /v2/.../manifests/<digest>`
- Confirmation dialogs for every destructive action
- Registry **health checks** (`healthy` / `unhealthy` / `unauthorized` / `unreachable`)

### Ops & governance
- Dashboard overview (repos, tags, status, recent activity)
- **Audit log** (login, registry changes, deletes, GC, user admin)
- User management (create / disable / delete / reset password)
- **Garbage collection** (local registry mode) with **dry-run**
- **Retention cleanup preview** (no automatic deletes in v1)
- Optional **Swagger / OpenAPI** UI

### Deployment
- Single multi-stage Docker image
- SQLite on a named volume (`/data/regpilot.db`)
- Fully Dockerized build & test — no local Go/Node toolchain required

---

## 🚀 Quick Start

```bash
git clone https://github.com/pouya9998/RegPilot.git
cd RegPilot
cp .env.example .env
docker compose up -d --build
```

Open:

```text
http://localhost:8080
```

Default login (change immediately):

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `change-me` |

> If port `8080` is already in use:
> ```bash
> HOST_PORT=8090 docker compose up -d --build
> ```
> Then open http://localhost:8090

---

## 🏗 Architecture

```text
┌────────────┐
│   Browser  │
└─────┬──────┘
      │
      ▼
┌─────────────────────────────────────┐
│           RegPilot container        │
│  ┌─────────────┐  ┌──────────────┐  │
│  │ React (Vite)│  │  Go API      │  │
│  │  static UI  │──│  Gin + JWT   │  │
│  └─────────────┘  └──────┬───────┘  │
│                          │          │
│                   ┌──────▼───────┐  │
│                   │ SQLite /data │  │
│                   └──────────────┘  │
└──────────────────────────┬──────────┘
                           │ Docker Registry HTTP API V2
                           ▼
                  ┌──────────────────┐
                  │ Docker Registry  │
                  │ (source of truth)│
                  └──────────────────┘
```

### Stack

| Layer | Tech |
|-------|------|
| Backend | Go, Gin, GORM, SQLite, JWT, bcrypt |
| Frontend | React, TypeScript, Vite, Tailwind, TanStack Query |
| Auth | JWT access tokens + role middleware |
| Docs | OpenAPI (`/swagger` when enabled) |
| Packaging | Multi-stage Dockerfile + Compose |

---

## 📁 Project layout

```text
RegPilot/
├── backend/                 # Go API (clean architecture)
│   ├── cmd/server/
│   └── internal/
│       ├── config/
│       ├── database/
│       ├── handlers/
│       ├── middleware/
│       ├── models/
│       ├── registry/        # Docker Registry V2 client + GC
│       ├── repositories/
│       ├── services/
│       └── utils/
├── frontend/                # React SPA
├── docs/openapi.json
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── .env.example
```

---

## ⚙️ Configuration

Copy `.env.example` → `.env` and set real secrets before production.

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | `production` / `development` | `production` |
| `APP_PORT` | Container listen port | `8080` |
| `HOST_PORT` | Host port mapping | `8080` |
| `DATABASE_PATH` | SQLite file path | `/data/regpilot.db` |
| `JWT_SECRET` | JWT signing key | **required** |
| `JWT_EXPIRATION` | Token TTL | `24h` |
| `ADMIN_USERNAME` | Bootstrap admin | `admin` |
| `ADMIN_PASSWORD` | Bootstrap password (bcrypt-hashed on first boot) | `change-me` |
| `ENCRYPTION_KEY` | Encrypts registry passwords (16+ chars) | **required** |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | `http://localhost:8080` |
| `ENABLE_SWAGGER` | Expose `/swagger` | `true` |
| `REGISTRY_CONTAINER_NAME` | Local registry container for GC | empty |
| `REGISTRY_GC_MODE` | `local` or `remote` | `local` |
| `LOGIN_RATE_LIMIT` | Max login attempts per window | `10` |
| `LOGIN_RATE_WINDOW` | Rate-limit window | `1m` |

Data persists in Docker volume **`regpilot-data`** → `/data`.

---

## 📦 Connecting a registry

1. Sign in as **admin**
2. Go to **Registries → Add Registry**
3. Set name + URL (example: `https://registry.example.com` or `http://192.168.1.100:5000`)
4. Optional username/password
5. Optional **Allow self-signed TLS** (this registry only)
6. **Test Connection**
7. Browse **Repositories**

For delete + GC to work usefully, enable delete on the registry:

```yaml
# registry config.yml
storage:
  delete:
    enabled: true
```

or:

```bash
-e REGISTRY_STORAGE_DELETE_ENABLED=true
```

---

## 🔐 Authentication & roles

| Role | Can do |
|------|--------|
| **ADMIN** | Registries CRUD, delete images, GC, users, audit logs |
| **VIEWER** | Read-only browse / inspect |

- Unauthenticated users → `/login`
- VIEWER cannot delete images or change configuration

---

## 🗑 Garbage Collection

> ⚠️ GC permanently removes unreferenced blobs. Always **Dry Run** first. Avoid push/delete traffic during GC.

Docker Registry HTTP API V2 **does not** expose a remote GC endpoint.

### Local GC (supported)

RegPilot runs a **fixed**, controlled command only:

```text
docker exec <REGISTRY_CONTAINER_NAME> \
  registry garbage-collect /etc/docker/registry/config.yml [--dry-run]
```

**Setup:**

1. Find the registry container name:
   ```bash
   docker ps --format '{{.Names}}\t{{.Image}}' | grep -i registry
   ```
2. In `.env`:
   ```env
   REGISTRY_GC_MODE=local
   REGISTRY_CONTAINER_NAME=registry
   ```
3. In `docker-compose.yml`, mount the Docker socket:
   ```yaml
   volumes:
     - regpilot-data:/data
     - /var/run/docker.sock:/var/run/docker.sock:ro
   ```
4. Restart:
   ```bash
   docker compose up -d
   ```
5. In the UI: **Dry Run** → review → **Run GC**

### Remote registries

If the registry is not a local container, Local GC cannot run from RegPilot. Run garbage collection on the registry host yourself.

---

## 🧹 Retention cleanup preview

Admins can preview a simple retention policy:

- Keep latest **N** tags
- Optionally delete tags older than **X** days

v1 returns a **preview only** (counts + estimated reclaimable size). Nothing is auto-deleted without explicit future confirmation flows.

---

## 🖥 Frontend pages

| Path | Description |
|------|-------------|
| `/login` | Authentication |
| `/dashboard` | Overview + activity |
| `/registries` | Registry list |
| `/registries/:id` | Health, GC, retention preview |
| `/registries/:id/repositories` | Repo browser |
| `/registries/:id/repositories/:repository` | Tags + delete |
| `/users` | User admin (ADMIN) |
| `/audit-logs` | Audit trail (ADMIN) |
| `/settings` | Account / security notes |

---

## 📘 API documentation

With `ENABLE_SWAGGER=true`:

| Resource | URL |
|----------|-----|
| Swagger UI | `/swagger` |
| OpenAPI JSON | `/api/openapi.json` |

Selected endpoints:

```text
POST   /api/auth/login
GET    /api/dashboard
GET    /api/registries
POST   /api/registries
POST   /api/registries/:id/test
GET    /api/registries/:id/health
GET    /api/registries/:id/repositories
GET    /api/registries/:id/tags?repository=
GET    /api/registries/:id/manifests?repository=&reference=
DELETE /api/registries/:id/images
POST   /api/registries/:id/gc
POST   /api/registries/:id/cleanup/preview
GET    /api/users
GET    /api/audit-logs
```

Consistent response shape:

```json
{ "data": {} }
```

```json
{ "error": { "code": "REGISTRY_UNREACHABLE", "message": "Unable to connect to registry" } }
```

---

## 🛠 Development

Fully Dockerized:

```bash
cp .env.example .env
make dev            # compose up --build
make test           # Go + frontend tests in containers
make lint
make docker-build
make docker-run
```

### Makefile targets

| Target | Action |
|--------|--------|
| `make dev` | Build & run with Compose |
| `make build` | Build image |
| `make test` | Backend + frontend tests |
| `make lint` | Typecheck / vet |
| `make docker-build` | Build image |
| `make docker-run` | Run detached |
| `make migrate` | Info — AutoMigrate on startup |

---

## ✅ Testing

```bash
make test
```

Covered areas include:

1. Login / invalid login  
2. Admin vs VIEWER authorization  
3. Add registry (password never leaked)  
4. Registry client: catalog, tags, manifest, delete  
5. Crypto / JWT helpers  
6. Audit log creation  
7. Frontend auth flow (mocked)

---

## 🔒 Security

RegPilot defaults to secure practices:

- bcrypt password hashing (plaintext never stored)
- JWT auth on protected routes
- Role-based authorization middleware
- AES encryption for registry passwords (`ENCRYPTION_KEY`)
- Credentials omitted from API responses
- Login rate limiting
- Security headers (CSP, `X-Frame-Options`, etc.)
- Input validation + GORM parameterized queries
- No arbitrary shell / filesystem access from user input
- Controlled GC command only

**Before production:**

1. Set strong `JWT_SECRET`, `ADMIN_PASSWORD`, `ENCRYPTION_KEY`
2. Restrict `CORS_ALLOWED_ORIGINS`
3. Set `ENABLE_SWAGGER=false` unless needed
4. Terminate TLS in front of RegPilot
5. Mount Docker socket **only** if Local GC is required

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

---

## 🧭 Roadmap (not in v1)

Interfaces are shaped so future providers can be added without rewriting the core UI:

- Harbor  
- GitLab Container Registry  
- Nexus Repository  
- AWS ECR / Google Artifact Registry / Azure ACR  

v1 implements **Docker Registry HTTP API V2 only**.

---

## 🤝 Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md).

1. Fork → feature branch  
2. Keep changes focused  
3. Run `make test` and `make docker-build`  
4. Open a PR  

---

## 📄 License

[MIT](LICENSE) © RegPilot contributors

---

<div align="center">

**RegPilot** — manage your Docker Registry without the bloat.

[Report a bug](https://github.com/pouya9998/RegPilot/issues) · [Request a feature](https://github.com/pouya9998/RegPilot/issues)

</div>
