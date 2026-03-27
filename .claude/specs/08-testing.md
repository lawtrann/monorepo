# 08 — Testing

---

## Strategy

| Layer | Approach | Tools |
|---|---|---|
| Domain entities | Pure unit tests | testify assertions |
| Use cases | Mocked repos | testify/mock + goshared/mock/ |
| Repos / RLS | Integration against real Postgres | testcontainers-go |
| Interceptors | Unit tests (error mapping, auth) | testify |
| goshared generics | Unit + integration | testify + testcontainers-go |
| CDC / Outbox | E2E with docker-compose | docker-compose test env |

---

## Unit Tests

### Domain Entities

```go
func TestBook_Rename(t *testing.T) {
    book := &domain.Book{ID: uuid.New(), Name: "Old"}
    err := book.Rename("New Name")
    assert.NoError(t, err)
    assert.Equal(t, "New Name", book.Name)
}

func TestBook_Rename_Empty(t *testing.T) {
    book := &domain.Book{ID: uuid.New(), Name: "Old"}
    err := book.Rename("")
    var invalid *apperr.InvalidInput
    assert.ErrorAs(t, err, &invalid)
}
```

### Error Interceptor

```go
func TestErrorInterceptor_NotFound(t *testing.T) {
    handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        return nil, &apperr.NotFound{Description: "book not found"}
    }
    intercepted := interceptor.ErrorInterceptor(handler)
    _, err := intercepted(ctx, req)

    var connErr *connect.Error
    assert.ErrorAs(t, err, &connErr)
    assert.Equal(t, connect.CodeNotFound, connErr.Code())
}

func TestErrorInterceptor_WrappedError(t *testing.T) {
    original := fmt.Errorf("db error")
    handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        return nil, &apperr.NotFound{Description: "book not found", Err: original}
    }
    intercepted := interceptor.ErrorInterceptor(handler)
    _, err := intercepted(ctx, req)

    var connErr *connect.Error
    assert.ErrorAs(t, err, &connErr)
    assert.Equal(t, connect.CodeNotFound, connErr.Code())
}
```

### Reflection

```go
func TestReflect_SkipsTenantColumn(t *testing.T) {
    row := &sqlgen.Book{BookID: uuid.New(), Name: "Test", Tenant: "lms"}
    cols, vals := structToColumnsAndValues(row)
    assert.NotContains(t, cols, "tenant")
    assert.NotContains(t, cols, "created_at")
}

func TestUpdateFields_MapsGoFieldToDbTag(t *testing.T) {
    row := &sqlgen.Book{Name: "Updated"}
    cols, vals := structToColumnsAndValuesForFields(row, []string{"Name"})
    assert.Equal(t, []string{"name"}, cols)
}
```

---

## Integration Tests

### RLS Isolation

```go
func TestRLS_CrossTenantIsolation(t *testing.T) {
    pool := setupTestDB(t)  // testcontainers-go

    ctxA := tenant.WithContext(ctx, "lms")
    repoA := infra.NewBookRepo(pool)
    repoA.Create(ctxA, &domain.Book{ID: uuid.New(), Name: "Book A"})

    ctxB := tenant.WithContext(ctx, "other-school")
    repoB := infra.NewBookRepo(pool)
    page, _ := repoB.List(ctxB, repo.ListFilter{Limit: 100})
    assert.Empty(t, page.Items)  // tenant B sees nothing from tenant A
}

func TestRLS_InsertAutoFillsTenant(t *testing.T) {
    pool := setupTestDB(t)
    ctxA := tenant.WithContext(ctx, "lms")
    bookRepo := infra.NewBookRepo(pool)
    bookRepo.Create(ctxA, &domain.Book{ID: uuid.New(), Name: "Book"})

    var tenant string
    pool.QueryRow(ctx, "SELECT tenant FROM eureka.books LIMIT 1").Scan(&tenant)
    assert.Equal(t, "lms", tenant)
}
```

---

## E2E Tests

### Full Flow

```
1. Login via Casdoor → get JWT
2. Create tenant (RPC + JWT)
3. Create location (RPC + JWT)
4. Create course (RPC + JWT)
5. Assign location to course
6. GetCourseWithLocations → verify
7. Create book in eureka (RPC + JWT)
8. Verify tenant isolation (second tenant sees nothing)
```

### CDC Flow

```
1. Create user in Casdoor → wait for CDC → assert mirror tables updated
2. Create course in mastermgmt → wait for CDC → assert eureka.courses updated
```

---

## CI Pipeline (GitHub Actions)

```yaml
jobs:
  lint:
    - golangci-lint
    - buf lint
  test:
    - go test ./pkg/goshared/...
    - go test ./app/mastermgmt/...
    - go test ./app/eureka/...
  build:
    - go build ./app/mastermgmt/cmd/server
    - go build ./app/mastermgmt/cmd/worker
    - go build ./app/eureka/cmd/server
    - go build ./app/eureka/cmd/worker
    - buf generate (verify no drift)
```

---

## Tasks

### Phase 10: Testing & CI

- [ ] **10.1** Unit tests — BaseRepo, MappedRepo, reflection, UpdateFields, pagination
- [ ] **10.2** Unit tests — domain entities, collection types, business logic
- [ ] **10.3** Unit tests — use cases with mocked repos (goshared/mock/)
- [ ] **10.4** Unit tests — error interceptor (apperr → ConnectRPC codes, wrapped errors)
- [ ] **10.5** Integration — repos against Postgres (testcontainers-go)
- [ ] **10.6** Integration — RLS: cross-tenant isolation, INSERT auto-fills tenant
- [ ] **10.7** Integration — JWT validation with Casdoor-issued tokens
- [ ] **10.8** E2E — CDC: Casdoor user → mirror tables
- [ ] **10.9** E2E — CDC: mastermgmt course → eureka mirror
- [ ] **10.10** E2E — full flow: login → tenant → location → course → assign → book → isolation
- [ ] **10.11** CI pipeline (GitHub Actions): lint, test, build
