# 01 — Monorepo Foundation

---

## Monorepo Structure

```
monorepo/
├── deployment/                       # Root-level — Terraform ready
│   ├── mastermgmt/
│   │   ├── local/
│   │   │   ├── config.yaml
│   │   │   ├── secrets.yaml          # SOPS+age encrypted
│   │   │   └── .sops.yaml
│   │   ├── staging/
│   │   ├── preprod/
│   │   └── prod/
│   ├── eureka/
│   │   ├── local/ staging/ preprod/ prod/
│   └── casdoor/
│       ├── local/ staging/ preprod/ prod/
│
├── docs/                             # Architecture & phase documents
│
├── pkg/
│   ├── goshared/                     # Common Go library (see 02-goshared.md)
│   ├── go/                           # Generated proto (Go)
│   │   ├── mastermgmt/v1/
│   │   └── eureka/v1/
│   └── ts/                           # Generated proto (TypeScript)
│       ├── mastermgmt/v1/
│       └── eureka/v1/
│
├── app/
│   ├── mastermgmt/
│   │   ├── proto/
│   │   │   ├── buf.yaml
│   │   │   ├── buf.gen.yaml
│   │   │   └── mastermgmt/v1/*.proto
│   │   ├── migration/
│   │   │   ├── alembic.ini
│   │   │   ├── alembic/
│   │   │   └── pyproject.toml
│   │   ├── cmd/
│   │   │   ├── server/main.go
│   │   │   └── worker/main.go
│   │   ├── internal/
│   │   │   ├── sqlgen/              # sqlc-generated models for mastermgmt schema
│   │   │   ├── tenant/
│   │   │   ├── location/
│   │   │   └── course/
│   │   ├── sqlc.yaml
│   │   ├── go.mod
│   │   ├── Dockerfile
│   │   ├── Dockerfile.worker
│   │   └── Makefile
│   │
│   ├── eureka/
│   │   ├── proto/
│   │   ├── migration/
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── sqlgen/              # sqlc-generated models for eureka schema
│   │   │   └── book/
│   │   ├── sqlc.yaml
│   │   ├── go.mod
│   │   ├── Dockerfile
│   │   ├── Dockerfile.worker
│   │   └── Makefile
│   │
│   └── web/
│       └── ...
│
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Tasks

- [ ] **0.1** Create `monorepo/` — `.gitignore`, `README.md`, root `Makefile`
- [ ] **0.2** Create `pkg/goshared/` Go module with `go.mod`, install common deps
- [ ] **0.3** Create `app/mastermgmt/` Go module — `go.mod` imports `pkg/goshared`
- [ ] **0.4** Create `app/eureka/` Go module — `go.mod` imports `pkg/goshared`
- [ ] **0.5** Create `app/mastermgmt/migration/` and `app/eureka/migration/` — Alembic setup per app
- [ ] **0.6** Create proto dirs — `app/mastermgmt/proto/`, `app/eureka/proto/`, with `buf.yaml` + `buf.gen.yaml`
- [ ] **0.7** Create `deployment/` at monorepo root with `mastermgmt/`, `eureka/`, `casdoor/` subdirs, each with `local/` env
- [ ] **0.8** Install tooling: Buf CLI, sqlc CLI, SOPS, age, air
- [ ] **0.9** Set up SOPS: generate age key, encrypt initial secrets.yaml per app
- [ ] **0.10** Create `app/web/` placeholder
- [ ] **0.11** Verify `buf lint` passes
