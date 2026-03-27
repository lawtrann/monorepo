package pgx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lawtrann/monorepo/pkg/goshared/db"
)

type contextKey string

const tenantKey contextKey = "tenant"

// PgxPool wraps pgxpool.Pool and implements db.Pool.
// It is tenant-aware: AfterConnect sets search_path, PrepareConn sets
// app.current_tenant from context, and AfterRelease resets it.
type PgxPool struct {
	pool *pgxpool.Pool
}

// NewPgxPool creates a PgxPool from the given config, wiring the tenant hooks.
func NewPgxPool(ctx context.Context, cfg db.Config) (*PgxPool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	if cfg.PoolSize > 0 {
		poolCfg.MaxConns = int32(cfg.PoolSize)
	}

	schema := cfg.Schema

	// AfterConnect: scope all queries to the tenant schema.
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		return err
	}

	// BeforeAcquire acts as PrepareConn: inject tenant slug from context.
	poolCfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		slug, ok := ctx.Value(tenantKey).(string)
		if !ok || slug == "" {
			return true
		}
		_, err := conn.Exec(ctx, "SET app.current_tenant = $1", slug)
		return err == nil
	}

	// AfterRelease: reset tenant context so the connection is clean for reuse.
	poolCfg.AfterRelease = func(conn *pgx.Conn) bool {
		_, err := conn.Exec(context.Background(), "RESET app.current_tenant")
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	return &PgxPool{pool: pool}, nil
}

// WithTenant returns a new context carrying the tenant slug for PrepareConn.
func WithTenant(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, tenantKey, slug)
}

// Acquire implements db.Pool.
func (p *PgxPool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return p.pool.Acquire(ctx)
}

// QueryRow implements db.Pool.
func (p *PgxPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Query implements db.Pool.
func (p *PgxPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// Exec implements db.Pool.
func (p *PgxPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

// Close implements db.Pool.
func (p *PgxPool) Close() {
	p.pool.Close()
}
