# 06 — CDC, Outbox & Workers

---

## CDC Architecture

Single PostgreSQL database (`platform`), one Debezium connector watches the WAL and routes changes to Kafka topics.

```
platform DB (single WAL)
│
├── casdoor.user        ──CDC──▶  cdc.casdoor.users
│                                    ├── mastermgmt-worker: upsert mastermgmt.users
│                                    └── eureka-worker: upsert eureka.users
│
├── mastermgmt.courses  ──CDC──▶  cdc.mastermgmt.courses
│                                    └── eureka-worker: upsert eureka.courses
│
├── mastermgmt.outbox   ──Outbox SMT──▶  outbox.event.<aggregatetype>
│
└── eureka.outbox       ──Outbox SMT──▶  outbox.event.<aggregatetype>
```

### Topic Naming

| Topic | Source | Consumers |
|---|---|---|
| `cdc.casdoor.users` | casdoor.user table | mastermgmt-worker, eureka-worker |
| `cdc.mastermgmt.courses` | mastermgmt.courses | eureka-worker |
| `outbox.event.tenant` | mastermgmt.outbox | workers |
| `outbox.event.course` | mastermgmt.outbox | workers |
| `outbox.event.book` | eureka.outbox | workers |

---

## Outbox Pattern

```
Use case → aggregate.AddEvent() → UoW.Commit()
  → BEGIN TX
  → INSERT INTO mastermgmt.tenants (...)
  → INSERT INTO mastermgmt.outbox (aggregatetype, aggregateid, type, payload)
  → COMMIT
  → Debezium reads outbox via WAL → Kafka → Workers consume
```

---

## Workers

**mastermgmt-worker** (`app/mastermgmt/cmd/worker/main.go`): consumes `cdc.casdoor.users` + `outbox.event.*`

**eureka-worker** (`app/eureka/cmd/worker/main.go`): consumes `cdc.casdoor.users` + `cdc.mastermgmt.courses` + `outbox.event.*`

Both use `pubsub/interfaces.Consumer` (implemented by `pubsub/franz.FranzConsumer`).

---

## Tasks

### Phase 8: CDC + Outbox + Workers

- [ ] **8.1** Add Zookeeper + Kafka + Debezium to `docker-compose.yml`
- [ ] **8.2** Configure Debezium source connector for `platform` DB
- [ ] **8.3** Configure CDC: `casdoor.user` → `cdc.casdoor.users` (selected columns)
- [ ] **8.4** Configure CDC: `mastermgmt.courses` → `cdc.mastermgmt.courses`
- [ ] **8.5** Configure Outbox Event Router for `mastermgmt.outbox`
- [ ] **8.6** Configure Outbox Event Router for `eureka.outbox`
- [ ] **8.7** `app/mastermgmt/cmd/worker/main.go` — Fx bootstrap, consume CDC + outbox
- [ ] **8.8** `app/eureka/cmd/worker/main.go` — Fx bootstrap, consume CDC + outbox
- [ ] **8.9** Dockerfiles for workers
- [ ] **8.10** Test CDC + outbox end-to-end
