package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX: sqlcが生成するクエリが受け付けるインターフェース
// pgxpool.Pool と pgx.Tx の両方がこれを満たす
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn interface{ RowsAffected() int64 }, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxManager: トランザクション管理ヘルパー
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager: TxManagerを生成
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// RunInTx: トランザクション内で関数を実行する
// 関数がエラーを返した場合はロールバック、成功した場合はコミットする
func (tm *TxManager) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Pool: 内部のpoolを返却（トランザクション外のクエリ用）
func (tm *TxManager) Pool() *pgxpool.Pool {
	return tm.pool
}
