# 07 — Docker & DevEx

Basic `db` + `casdoor` compose is set up in Phase 03. This phase expands to full 11-service compose.

---

## Full Docker Compose

| Service | Image | Ports | Purpose |
|---|---|---|---|
| `db` | `postgres:18` | 5432 | Single `platform` DB |
| `minio` | `minio/minio` | 9000, 9001 | Object storage |
| `casdoor` | `casbin/casdoor` | 8000 | IdP |
| `zookeeper` | `confluentinc/cp-zookeeper` | 2181 | Kafka coordination |
| `kafka` | `confluentinc/cp-kafka` | 9092 | Event streaming |
| `debezium` | `debezium/connect` | 8083 | CDC + Outbox relay |
| `mastermgmt` | Dockerfile | 8080 | API |
| `mastermgmt-worker` | Dockerfile.worker | — | CDC consumer |
| `eureka` | Dockerfile | 8081 | API |
| `eureka-worker` | Dockerfile.worker | — | CDC consumer |
| `web` | Next.js | 3000 | Frontend |

---

## Dockerfiles

4 total — multi-stage builds (Go build → distroless/alpine runtime):

- `app/mastermgmt/Dockerfile` + `Dockerfile.worker`
- `app/eureka/Dockerfile` + `Dockerfile.worker`

---

## Makefiles

### Root Makefile

```makefile
dev:                    ## docker-compose up
migrate-mastermgmt:     ## Run mastermgmt Alembic migrations
migrate-eureka:         ## Run eureka Alembic migrations
proto:                  ## buf generate for all apps
sqlc:                   ## sqlc generate for all apps
test:                   ## Run all tests
lint:                   ## golangci-lint + buf lint
```

### Per-App Makefiles

```makefile
run:                    ## air hot-reload
build:                  ## go build
test:                   ## go test ./...
sqlc:                   ## sqlc generate
```

---

## Config Loading

Each service reads from `deployment/<app>/local/`:

```yaml
# deployment/mastermgmt/local/config.yaml
server:
  port: 8080
db:
  host: db
  port: 5432
  database: platform
  schema: mastermgmt
casdoor:
  jwks_url: http://casdoor:8000/.well-known/jwks
```

Secrets in `secrets.yaml` (SOPS+age encrypted, committed).

---

## Tasks

### Phase 9: Full Compose + DevEx

- [ ] **9.1** Expand `docker-compose.yml` to 11 services (add minio, zk, kafka, debezium, workers, web)
- [ ] **9.2** Dockerfiles: server + worker per app (4 total, multi-stage)
- [ ] **9.3** Root `Makefile` (dev, migrate, sqlc, proto, test, lint)
- [ ] **9.4** Per-app Makefiles
- [ ] **9.5** `air` hot-reload config per service
- [ ] **9.6** Local config.yaml + secrets.yaml per app
