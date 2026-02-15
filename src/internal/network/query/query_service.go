package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// NetworkQueryService: Network BCの読み取り専用クエリサービス
type NetworkQueryService struct {
	q *sqlcgen.Queries
}

func NewNetworkQueryService(pool *pgxpool.Pool) *NetworkQueryService {
	return &NetworkQueryService{q: sqlcgen.New(pool)}
}

func (s *NetworkQueryService) GetLocation(ctx context.Context, id uuid.UUID) (sqlcgen.Location, error) {
	return s.q.GetLocationByID(ctx, id)
}

func (s *NetworkQueryService) GetLocationByUNLocode(ctx context.Context, code string) (sqlcgen.Location, error) {
	return s.q.GetLocationByUnLocode(ctx, pgtype.Text{String: code, Valid: true})
}

func (s *NetworkQueryService) GetLane(ctx context.Context, id uuid.UUID) (sqlcgen.Lane, error) {
	return s.q.GetLaneByID(ctx, id)
}

func (s *NetworkQueryService) GetStandardRoute(ctx context.Context, id uuid.UUID) (sqlcgen.StandardRoute, error) {
	return s.q.GetStandardRouteByID(ctx, id)
}

func (s *NetworkQueryService) ListStandardRoutes(ctx context.Context, shipperID uuid.UUID) ([]sqlcgen.StandardRoute, error) {
	return s.q.ListActiveStandardRoutesByShipper(ctx, shipperID)
}
