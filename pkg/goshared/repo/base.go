package repo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lawtrann/monorepo/pkg/goshared/apperr"
	"github.com/lawtrann/monorepo/pkg/goshared/db"
)

// BaseRepo provides generic CRUD operations for a single table Row type R with primary key type ID.
// It operates on db.Pool (interface) so it is testable without a real database.
type BaseRepo[R any, ID comparable] struct {
	pool  db.Pool
	table string
	pk    string
}

// NewBaseRepo constructs a BaseRepo for the given table and primary-key column name.
func NewBaseRepo[R any, ID comparable](pool db.Pool, table, pk string) *BaseRepo[R, ID] {
	return &BaseRepo[R, ID]{pool: pool, table: table, pk: pk}
}

// GetByID fetches a single row by primary key, returning apperr.NotFound when absent.
func (r *BaseRepo[R, ID]) GetByID(ctx context.Context, id ID) (*R, error) {
	q := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 AND deleted_at IS NULL", r.table, r.pk)
	rows, err := r.pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[R])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &apperr.NotFound{Description: fmt.Sprintf("%s not found", r.table), Err: err}
		}
		return nil, err
	}
	return &row, nil
}

// List returns a paginated result set.
//   - Cursor != nil  → keyset pagination (WHERE pk > cursor, LIMIT limit+1 to detect next page)
//   - Offset != nil  → offset pagination with a total count
//   - Otherwise      → plain LIMIT query
func (r *BaseRepo[R, ID]) List(ctx context.Context, f ListFilter) (*Page[R], error) {
	args := make([]any, 0, 4)
	conds := []string{"deleted_at IS NULL"}

	if f.Cursor != nil {
		args = append(args, *f.Cursor)
		conds = append(conds, fmt.Sprintf("%s > $%d", r.pk, len(args)))
	}

	whereSQL := "WHERE " + strings.Join(conds, " AND ")

	sortBy := r.pk
	if f.SortBy != "" {
		sortBy = f.SortBy
	}
	sortOrder := "ASC"
	if f.SortOrder == "DESC" {
		sortOrder = "DESC"
	}

	limit := f.Limit
	if limit <= 0 {
		limit = f.PageSize
	}
	if limit <= 0 {
		limit = 20
	}

	// Offset-based pagination includes a total count.
	if f.Offset != nil {
		var total int64
		countQ := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", r.table, whereSQL)
		if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
			return nil, err
		}

		queryArgs := append(append([]any{}, args...), *f.Offset, limit)
		q := fmt.Sprintf(
			"SELECT * FROM %s %s ORDER BY %s %s OFFSET $%d LIMIT $%d",
			r.table, whereSQL, sortBy, sortOrder,
			len(queryArgs)-1, len(queryArgs),
		)
		rows, err := r.pool.Query(ctx, q, queryArgs...)
		if err != nil {
			return nil, err
		}
		items, err := pgx.CollectRows(rows, pgx.RowToStructByName[R])
		if err != nil {
			return nil, err
		}
		return &Page[R]{Items: items, Total: &total}, nil
	}

	// Cursor / basic pagination: fetch limit+1 to detect whether a next page exists.
	args = append(args, limit+1)
	q := fmt.Sprintf(
		"SELECT * FROM %s %s ORDER BY %s %s LIMIT $%d",
		r.table, whereSQL, sortBy, sortOrder, len(args),
	)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[R])
	if err != nil {
		return nil, err
	}

	var nextCursor *uuid.UUID
	if int32(len(items)) > limit {
		items = items[:limit]
		nextCursor = extractUUIDByCol(items[len(items)-1], r.pk)
	}
	return &Page[R]{Items: items, NextCursor: nextCursor}, nil
}

// Create inserts a new row and returns the persisted record (with DB-generated fields populated).
func (r *BaseRepo[R, ID]) Create(ctx context.Context, row *R) (*R, error) {
	cols, vals := structToColumnsAndValues(row)
	if len(cols) == 0 {
		return nil, fmt.Errorf("no insertable columns on %T", row)
	}

	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	q := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		r.table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	rows, err := r.pool.Query(ctx, q, vals...)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[R])
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update replaces all mutable columns for an existing row, returning the updated record.
func (r *BaseRepo[R, ID]) Update(ctx context.Context, id ID, row *R) (*R, error) {
	cols, vals := structToColumnsAndValues(row)
	if len(cols) == 0 {
		return nil, fmt.Errorf("no updatable columns on %T", row)
	}

	setClauses := make([]string, len(cols))
	for i, col := range cols {
		setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}
	vals = append(vals, id)
	q := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d AND deleted_at IS NULL RETURNING *",
		r.table,
		strings.Join(setClauses, ", "),
		r.pk,
		len(vals),
	)

	rows, err := r.pool.Query(ctx, q, vals...)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[R])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &apperr.NotFound{Description: fmt.Sprintf("%s not found", r.table), Err: err}
		}
		return nil, err
	}
	return &result, nil
}

// UpdateFields performs a partial update using Go struct field names as a mask.
// Only the listed fields (mapped via db tags) are written to the database.
func (r *BaseRepo[R, ID]) UpdateFields(ctx context.Context, id ID, row *R, fields []string) (*R, error) {
	wantCols := fieldsToColumns(row, fields)
	if len(wantCols) == 0 {
		return r.GetByID(ctx, id)
	}

	wantSet := make(map[string]bool, len(wantCols))
	for _, c := range wantCols {
		wantSet[c] = true
	}

	allCols, allVals := structToColumnsAndValues(row)
	var filteredCols []string
	var filteredVals []any
	for i, c := range allCols {
		if wantSet[c] {
			filteredCols = append(filteredCols, c)
			filteredVals = append(filteredVals, allVals[i])
		}
	}
	if len(filteredCols) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses := make([]string, len(filteredCols))
	for i, col := range filteredCols {
		setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}
	filteredVals = append(filteredVals, id)
	q := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d AND deleted_at IS NULL RETURNING *",
		r.table,
		strings.Join(setClauses, ", "),
		r.pk,
		len(filteredVals),
	)

	rows, err := r.pool.Query(ctx, q, filteredVals...)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[R])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &apperr.NotFound{Description: fmt.Sprintf("%s not found", r.table), Err: err}
		}
		return nil, err
	}
	return &result, nil
}

// SoftDelete sets deleted_at = now() for the given row, returning apperr.NotFound if absent.
func (r *BaseRepo[R, ID]) SoftDelete(ctx context.Context, id ID) error {
	q := fmt.Sprintf(
		"UPDATE %s SET deleted_at = now() WHERE %s = $1 AND deleted_at IS NULL",
		r.table, r.pk,
	)
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return &apperr.NotFound{Description: fmt.Sprintf("%s not found", r.table)}
	}
	return nil
}

// extractUUIDByCol reads a uuid.UUID from the struct field whose db tag matches col.
// Returns nil if the field is not found or is not a uuid.UUID.
func extractUUIDByCol[R any](row R, col string) *uuid.UUID {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("db")
		if strings.Split(tag, ",")[0] == col {
			if uid, ok := v.Field(i).Interface().(uuid.UUID); ok {
				return &uid
			}
		}
	}
	return nil
}
