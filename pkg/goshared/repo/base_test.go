package repo

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lawtrann/monorepo/pkg/goshared/apperr"
	"github.com/lawtrann/monorepo/pkg/goshared/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- test row type ----

type baseTestRow struct {
	ID   uuid.UUID `db:"id"`
	Name string    `db:"name"`
}

// ---- mock Pool ----

type mockPool struct {
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

var _ db.Pool = (*mockPool)(nil)

func (m *mockPool) Acquire(ctx context.Context) (*pgxpool.Conn, error) { return nil, nil }
func (m *mockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFn == nil {
		return nil, errors.New("queryFn not set")
	}
	return m.queryFn(ctx, sql, args...)
}
func (m *mockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFn == nil {
		return &mockRow{err: errors.New("queryRowFn not set")}
	}
	return m.queryRowFn(ctx, sql, args...)
}
func (m *mockPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.execFn == nil {
		return pgconn.CommandTag{}, errors.New("execFn not set")
	}
	return m.execFn(ctx, sql, args...)
}
func (m *mockPool) Close() {}

// ---- mock Row (for QueryRow / COUNT queries) ----

type mockRow struct {
	values []any
	err    error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i >= len(r.values) {
			break
		}
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(r.values[i]))
	}
	return nil
}

// ---- mock Rows ----

// rowScanner matches pgx.RowScanner without importing the unexported type.
type rowScanner interface {
	ScanRow(rows pgx.Rows) error
}

type mockRows struct {
	fieldDescs []pgconn.FieldDescription
	rowData    [][]any
	idx        int
	queryErr   error // error returned by Query itself
}

var _ pgx.Rows = (*mockRows)(nil)

func newMockRows(cols []string, rows [][]any) *mockRows {
	descs := make([]pgconn.FieldDescription, len(cols))
	for i, col := range cols {
		descs[i] = pgconn.FieldDescription{Name: col}
	}
	return &mockRows{fieldDescs: descs, rowData: rows}
}

func (m *mockRows) Close()                                       {}
func (m *mockRows) Err() error                                   { return nil }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return m.fieldDescs }
func (m *mockRows) Next() bool {
	if m.idx < len(m.rowData) {
		m.idx++
		return true
	}
	return false
}
func (m *mockRows) Scan(dest ...any) error {
	// pgx.RowToStructByName passes a *namedStructRowScanner which implements rowScanner.
	// Let it call ScanRow so it can match fields by name, then we handle the second
	// positional Scan call below.
	if len(dest) == 1 {
		if rs, ok := dest[0].(rowScanner); ok {
			return rs.ScanRow(m)
		}
	}
	// Positional assignment: dest[i] is a pointer to a struct field in FieldDescriptions order.
	row := m.rowData[m.idx-1]
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}
func (m *mockRows) Values() ([]any, error)    { return m.rowData[m.idx-1], nil }
func (m *mockRows) RawValues() [][]byte       { return nil }
func (m *mockRows) Conn() *pgx.Conn          { return nil }

// ---- helper ----

func newRepo(pool db.Pool) *BaseRepo[baseTestRow, uuid.UUID] {
	return NewBaseRepo[baseTestRow, uuid.UUID](pool, "items", "id")
}

var testCols = []string{"id", "name"}

// ============================================================
// BaseRepo tests
// ============================================================

func TestGetByID_Found(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "alice"}}), nil
		},
	}
	repo := newRepo(pool)
	row, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, row.ID)
	assert.Equal(t, "alice", row.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, nil), nil // zero rows
		},
	}
	repo := newRepo(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	var notFound *apperr.NotFound
	assert.True(t, errors.As(err, &notFound))
}

func TestGetByID_DBError(t *testing.T) {
	dbErr := errors.New("connection refused")
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return nil, dbErr
		},
	}
	repo := newRepo(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, dbErr)
}

func TestList_Basic(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{
				{id1, "alice"},
				{id2, "bob"},
			}), nil
		},
	}
	repo := newRepo(pool)
	page, err := repo.List(context.Background(), ListFilter{Limit: 5})
	require.NoError(t, err)
	assert.Len(t, page.Items, 2)
	assert.Nil(t, page.NextCursor)
	assert.Nil(t, page.Total)
}

func TestList_NextCursor(t *testing.T) {
	// With Limit=2 and 3 rows returned, the repo detects a next page.
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{
				{ids[0], "a"},
				{ids[1], "b"},
				{ids[2], "c"}, // extra row signaling next page
			}), nil
		},
	}
	repo := newRepo(pool)
	page, err := repo.List(context.Background(), ListFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page.Items, 2)
	require.NotNil(t, page.NextCursor)
	assert.Equal(t, ids[1], *page.NextCursor) // last kept item's ID
}

func TestList_OffsetPagination(t *testing.T) {
	id := uuid.New()
	offset := int32(0)
	total := int64(42)
	pool := &mockPool{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{values: []any{total}}
		},
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "alice"}}), nil
		},
	}
	repo := newRepo(pool)
	page, err := repo.List(context.Background(), ListFilter{Limit: 10, Offset: &offset})
	require.NoError(t, err)
	assert.Len(t, page.Items, 1)
	require.NotNil(t, page.Total)
	assert.Equal(t, total, *page.Total)
}

func TestCreate(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "new-item"}}), nil
		},
	}
	repo := newRepo(pool)
	row, err := repo.Create(context.Background(), &baseTestRow{ID: id, Name: "new-item"})
	require.NoError(t, err)
	assert.Equal(t, id, row.ID)
	assert.Equal(t, "new-item", row.Name)
}

func TestUpdate_Found(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "updated"}}), nil
		},
	}
	repo := newRepo(pool)
	row, err := repo.Update(context.Background(), id, &baseTestRow{ID: id, Name: "updated"})
	require.NoError(t, err)
	assert.Equal(t, "updated", row.Name)
}

func TestUpdate_NotFound(t *testing.T) {
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, nil), nil
		},
	}
	repo := newRepo(pool)
	_, err := repo.Update(context.Background(), uuid.New(), &baseTestRow{Name: "x"})
	var notFound *apperr.NotFound
	assert.True(t, errors.As(err, &notFound))
}

func TestUpdateFields(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "partial"}}), nil
		},
	}
	repo := newRepo(pool)
	row, err := repo.UpdateFields(context.Background(), id, &baseTestRow{ID: id, Name: "partial"}, []string{"Name"})
	require.NoError(t, err)
	assert.Equal(t, "partial", row.Name)
}

func TestUpdateFields_EmptyFields_FallsBackToGetByID(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "unchanged"}}), nil
		},
	}
	repo := newRepo(pool)
	// No valid fields → falls back to GetByID
	row, err := repo.UpdateFields(context.Background(), id, &baseTestRow{ID: id, Name: "unchanged"}, []string{})
	require.NoError(t, err)
	assert.Equal(t, "unchanged", row.Name)
}

func TestSoftDelete_Found(t *testing.T) {
	pool := &mockPool{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := newRepo(pool)
	err := repo.SoftDelete(context.Background(), uuid.New())
	assert.NoError(t, err)
}

func TestSoftDelete_NotFound(t *testing.T) {
	pool := &mockPool{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}
	repo := newRepo(pool)
	err := repo.SoftDelete(context.Background(), uuid.New())
	var notFound *apperr.NotFound
	assert.True(t, errors.As(err, &notFound))
}

func TestSoftDelete_DBError(t *testing.T) {
	dbErr := errors.New("db down")
	pool := &mockPool{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, dbErr
		},
	}
	repo := newRepo(pool)
	err := repo.SoftDelete(context.Background(), uuid.New())
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================
// MappedRepo tests
// ============================================================

type testEntity struct {
	ID   uuid.UUID
	Name string
}

func toDomainFn(r *baseTestRow) *testEntity {
	return &testEntity{ID: r.ID, Name: r.Name}
}

func toRowFn(e *testEntity) *baseTestRow {
	return &baseTestRow{ID: e.ID, Name: e.Name}
}

func newMappedRepo(pool db.Pool) *MappedRepo[testEntity, baseTestRow, uuid.UUID] {
	base := NewBaseRepo[baseTestRow, uuid.UUID](pool, "items", "id")
	return NewMappedRepo[testEntity, baseTestRow, uuid.UUID](base, toDomainFn, toRowFn)
}

func TestMappedRepo_GetByID(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "mapped"}}), nil
		},
	}
	repo := newMappedRepo(pool)
	entity, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, entity.ID)
	assert.Equal(t, "mapped", entity.Name)
}

func TestMappedRepo_GetByID_NotFound(t *testing.T) {
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, nil), nil
		},
	}
	repo := newMappedRepo(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	var notFound *apperr.NotFound
	assert.True(t, errors.As(err, &notFound))
}

func TestMappedRepo_List(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{
				{ids[0], "first"},
				{ids[1], "second"},
			}), nil
		},
	}
	repo := newMappedRepo(pool)
	page, err := repo.List(context.Background(), ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, page.Items, 2)
	assert.Equal(t, "first", page.Items[0].Name)
	assert.Equal(t, "second", page.Items[1].Name)
}

func TestMappedRepo_Create(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "created"}}), nil
		},
	}
	repo := newMappedRepo(pool)
	entity, err := repo.Create(context.Background(), &testEntity{ID: id, Name: "created"})
	require.NoError(t, err)
	assert.Equal(t, id, entity.ID)
	assert.Equal(t, "created", entity.Name)
}

func TestMappedRepo_Update(t *testing.T) {
	id := uuid.New()
	pool := &mockPool{
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return newMockRows(testCols, [][]any{{id, "updated-entity"}}), nil
		},
	}
	repo := newMappedRepo(pool)
	entity, err := repo.Update(context.Background(), id, &testEntity{ID: id, Name: "updated-entity"})
	require.NoError(t, err)
	assert.Equal(t, "updated-entity", entity.Name)
}

func TestMappedRepo_SoftDelete(t *testing.T) {
	pool := &mockPool{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}
	repo := newMappedRepo(pool)
	err := repo.SoftDelete(context.Background(), uuid.New())
	assert.NoError(t, err)
}
