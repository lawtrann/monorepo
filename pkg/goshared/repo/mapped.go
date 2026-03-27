package repo

import "context"

// MappedRepo wraps BaseRepo[R, ID] and translates between a domain entity type E and a DB row
// type R using caller-supplied toDomain and toRow functions.
type MappedRepo[E any, R any, ID comparable] struct {
	base     *BaseRepo[R, ID]
	toDomain func(*R) *E
	toRow    func(*E) *R
}

// NewMappedRepo constructs a MappedRepo backed by an existing BaseRepo.
func NewMappedRepo[E any, R any, ID comparable](
	base *BaseRepo[R, ID],
	toDomain func(*R) *E,
	toRow func(*E) *R,
) *MappedRepo[E, R, ID] {
	return &MappedRepo[E, R, ID]{base: base, toDomain: toDomain, toRow: toRow}
}

// GetByID fetches a domain entity by primary key.
func (m *MappedRepo[E, R, ID]) GetByID(ctx context.Context, id ID) (*E, error) {
	row, err := m.base.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return m.toDomain(row), nil
}

// List returns a paginated page of domain entities.
func (m *MappedRepo[E, R, ID]) List(ctx context.Context, f ListFilter) (*Page[E], error) {
	page, err := m.base.List(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]E, len(page.Items))
	for i := range page.Items {
		items[i] = *m.toDomain(&page.Items[i])
	}
	return &Page[E]{Items: items, NextCursor: page.NextCursor, Total: page.Total}, nil
}

// Create inserts a domain entity and returns the persisted domain entity.
func (m *MappedRepo[E, R, ID]) Create(ctx context.Context, entity *E) (*E, error) {
	row, err := m.base.Create(ctx, m.toRow(entity))
	if err != nil {
		return nil, err
	}
	return m.toDomain(row), nil
}

// Update replaces all mutable columns for an existing domain entity.
func (m *MappedRepo[E, R, ID]) Update(ctx context.Context, id ID, entity *E) (*E, error) {
	row, err := m.base.Update(ctx, id, m.toRow(entity))
	if err != nil {
		return nil, err
	}
	return m.toDomain(row), nil
}

// UpdateFields performs a partial update using Go struct field names as a mask.
func (m *MappedRepo[E, R, ID]) UpdateFields(ctx context.Context, id ID, entity *E, fields []string) (*E, error) {
	row, err := m.base.UpdateFields(ctx, id, m.toRow(entity), fields)
	if err != nil {
		return nil, err
	}
	return m.toDomain(row), nil
}

// SoftDelete sets deleted_at = now() for the given entity.
func (m *MappedRepo[E, R, ID]) SoftDelete(ctx context.Context, id ID) error {
	return m.base.SoftDelete(ctx, id)
}
