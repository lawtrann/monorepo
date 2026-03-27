package repo

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type reflectTestRow struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	TenantID  string    `db:"tenant"`
	CreatedAt string    `db:"created_at"`
	UpdatedAt string    `db:"updated_at"`
	DeletedAt string    `db:"deleted_at"`
	NoTag     string
	Ignored   string `db:"-"`
}

func TestStructToColumnsAndValues_ReturnsNonSkipped(t *testing.T) {
	id := uuid.New()
	row := &reflectTestRow{ID: id, Name: "alice", TenantID: "t1"}

	cols, vals := structToColumnsAndValues(row)

	assert.Equal(t, []string{"id", "name"}, cols)
	assert.Equal(t, []any{id, "alice"}, vals)
}

func TestStructToColumnsAndValues_SkipsTimestampsAndTenant(t *testing.T) {
	row := &reflectTestRow{TenantID: "x", CreatedAt: "now", UpdatedAt: "now", DeletedAt: "now"}
	cols, _ := structToColumnsAndValues(row)
	for _, c := range cols {
		assert.NotContains(t, []string{"tenant", "created_at", "updated_at", "deleted_at"}, c)
	}
}

func TestStructToColumnsAndValues_SkipsNoTagAndDashTag(t *testing.T) {
	row := &reflectTestRow{NoTag: "skip", Ignored: "skip"}
	cols, _ := structToColumnsAndValues(row)
	for _, c := range cols {
		assert.NotEqual(t, "NoTag", c)
		assert.NotEqual(t, "-", c)
	}
}

func TestFieldToColumn_Found(t *testing.T) {
	row := &reflectTestRow{}
	col, ok := fieldToColumn(row, "Name")
	assert.True(t, ok)
	assert.Equal(t, "name", col)
}

func TestFieldToColumn_NotFound(t *testing.T) {
	row := &reflectTestRow{}
	_, ok := fieldToColumn(row, "NonExistent")
	assert.False(t, ok)
}

func TestFieldToColumn_IgnoresDashTag(t *testing.T) {
	row := &reflectTestRow{}
	_, ok := fieldToColumn(row, "Ignored")
	assert.False(t, ok)
}

func TestFieldsToColumns_MapsAndFilters(t *testing.T) {
	row := &reflectTestRow{}
	cols := fieldsToColumns(row, []string{"ID", "Name", "NonExistent", "TenantID"})
	// ID→id, Name→name; NonExistent dropped; TenantID skipped (skippedColumns)
	assert.Equal(t, []string{"id", "name"}, cols)
}
