package parser

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
)

// LocationResolver: CSV中の拠点名/UN/LOCODEをLocationIDに変換
type LocationResolver interface {
	Resolve(ctx context.Context, nameOrCode string) (route.LocationID, error)
}

// PostgresLocationResolver: PostgreSQLを使用したLocationResolver実装
type PostgresLocationResolver struct {
	pool  *pgxpool.Pool
	mu    sync.RWMutex
	cache map[string]route.LocationID
}

// NewPostgresLocationResolver: PostgresLocationResolverを生成
func NewPostgresLocationResolver(pool *pgxpool.Pool) *PostgresLocationResolver {
	return &PostgresLocationResolver{
		pool:  pool,
		cache: make(map[string]route.LocationID),
	}
}

// Resolve: 名前またはUN/LOCODEからLocationIDを解決
// 解決順: 1. UN/LOCODE完全一致 → 2. Name完全一致(case-insensitive) → 3. Name部分一致
func (r *PostgresLocationResolver) Resolve(ctx context.Context, nameOrCode string) (route.LocationID, error) {
	nameOrCode = strings.TrimSpace(nameOrCode)
	if nameOrCode == "" {
		return route.LocationID(uuid.Nil), fmt.Errorf("empty location name or code")
	}

	// キャッシュチェック
	cacheKey := strings.ToUpper(nameOrCode)
	r.mu.RLock()
	if id, ok := r.cache[cacheKey]; ok {
		r.mu.RUnlock()
		return id, nil
	}
	r.mu.RUnlock()

	var locID uuid.UUID

	// 1. UN/LOCODE完全一致
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM locations WHERE un_locode = $1`,
		strings.ToUpper(nameOrCode)).Scan(&locID)
	if err == nil {
		return r.cacheAndReturn(cacheKey, locID), nil
	}
	if err != pgx.ErrNoRows {
		return route.LocationID(uuid.Nil), fmt.Errorf("query UN/LOCODE: %w", err)
	}

	// 2. Name完全一致 (case-insensitive)
	err = r.pool.QueryRow(ctx,
		`SELECT id FROM locations WHERE UPPER(name) = UPPER($1) LIMIT 1`,
		nameOrCode).Scan(&locID)
	if err == nil {
		return r.cacheAndReturn(cacheKey, locID), nil
	}
	if err != pgx.ErrNoRows {
		return route.LocationID(uuid.Nil), fmt.Errorf("query name exact: %w", err)
	}

	// 3. Name部分一致 (case-insensitive)
	err = r.pool.QueryRow(ctx,
		`SELECT id FROM locations WHERE UPPER(name) LIKE '%' || UPPER($1) || '%' ORDER BY LENGTH(name) ASC LIMIT 1`,
		nameOrCode).Scan(&locID)
	if err == nil {
		return r.cacheAndReturn(cacheKey, locID), nil
	}
	if err != pgx.ErrNoRows {
		return route.LocationID(uuid.Nil), fmt.Errorf("query name partial: %w", err)
	}

	return route.LocationID(uuid.Nil), fmt.Errorf("location not found: %q", nameOrCode)
}

func (r *PostgresLocationResolver) cacheAndReturn(key string, id uuid.UUID) route.LocationID {
	locID := route.LocationID(id)
	r.mu.Lock()
	r.cache[key] = locID
	r.mu.Unlock()
	return locID
}
