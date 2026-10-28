package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxManager implements ports.TxManager.
type TxManager struct {
	db *DB
}

func NewTxManager(db *DB) *TxManager { return &TxManager{db: db} }

// WithTx runs fn in a transaction and commits if it returns nil. A call made
// inside an existing transaction joins it instead of opening another.
func (m *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		// Use a fresh context: ctx may already be cancelled, and the
		// transaction must still be released.
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && err == nil {
			err = fmt.Errorf("rolling back transaction: %w", rbErr)
		}
	}()

	if err = fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	committed = true
	return nil
}
