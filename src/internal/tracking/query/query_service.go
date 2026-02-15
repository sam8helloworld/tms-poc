package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// TrackingQueryService: Tracking BCの読み取り専用クエリサービス
type TrackingQueryService struct {
	q *sqlcgen.Queries
}

func NewTrackingQueryService(pool *pgxpool.Pool) *TrackingQueryService {
	return &TrackingQueryService{q: sqlcgen.New(pool)}
}

func (s *TrackingQueryService) GetTrackingUnit(ctx context.Context, id uuid.UUID) (sqlcgen.TrackingUnit, error) {
	return s.q.GetTrackingUnitByID(ctx, id)
}

func (s *TrackingQueryService) GetTrackingUnitByNumber(ctx context.Context, number string) (sqlcgen.TrackingUnit, error) {
	return s.q.GetTrackingUnitByNumber(ctx, number)
}
