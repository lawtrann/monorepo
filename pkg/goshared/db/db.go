package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the database connection pool interface used by repositories.
type Pool interface {
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

// Config holds the database connection configuration.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Schema   string
	PoolSize int
}

// UnitOfWork wraps a transaction for atomic multi-repo operations.
type UnitOfWork interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
