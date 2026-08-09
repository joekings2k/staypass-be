package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
)

// TransactionRunner runs a unit of work inside a single database transaction.
// The callback receives a *dbgen.Queries bound to that transaction, so every
// query it issues commits or rolls back together.
type TransactionRunner interface {
	Run(ctx context.Context, fn func(q *dbgen.Queries) error) error
}

type txRunner struct {
	pool *pgxpool.Pool
}

func NewTransactionRunner(pool *pgxpool.Pool) TransactionRunner {
	return &txRunner{pool: pool}
}

func (r *txRunner) Run(ctx context.Context, fn func(q *dbgen.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	// Rollback is a no-op once the tx has been committed, so this safely
	// covers the error path, an early return inside fn, and a panic.
	defer tx.Rollback(ctx)

	q := dbgen.New(tx)
	if err := fn(q); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
