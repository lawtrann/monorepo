package repo

import "github.com/google/uuid"

// ListFilter controls pagination and sorting for list queries.
type ListFilter struct {
	Cursor    *uuid.UUID
	Limit     int32
	Offset    *int32
	PageSize  int32
	SortBy    string
	SortOrder string // "ASC" | "DESC"
}

// Page holds a paginated result set.
type Page[T any] struct {
	Items      []T
	NextCursor *uuid.UUID
	Total      *int64 // only populated for offset pagination
}
