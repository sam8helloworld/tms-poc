package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// RateQueryService: Rate BCの読み取り専用クエリサービス
type RateQueryService struct {
	q *sqlcgen.Queries
}

func NewRateQueryService(pool *pgxpool.Pool) *RateQueryService {
	return &RateQueryService{q: sqlcgen.New(pool)}
}

func (s *RateQueryService) GetRate(ctx context.Context, id uuid.UUID) (sqlcgen.Rate, error) {
	return s.q.GetRateByID(ctx, id)
}

func (s *RateQueryService) ListRatesByShipper(ctx context.Context, shipperID uuid.UUID) ([]sqlcgen.Rate, error) {
	return s.q.ListRatesByShipper(ctx, shipperID)
}
