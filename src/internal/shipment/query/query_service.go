package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// ShipmentQueryService: Shipment BCの読み取り専用クエリサービス
type ShipmentQueryService struct {
	q *sqlcgen.Queries
}

func NewShipmentQueryService(pool *pgxpool.Pool) *ShipmentQueryService {
	return &ShipmentQueryService{q: sqlcgen.New(pool)}
}

func (s *ShipmentQueryService) GetShipment(ctx context.Context, id uuid.UUID) (sqlcgen.Shipment, error) {
	return s.q.GetShipmentByID(ctx, id)
}

func (s *ShipmentQueryService) GetShipmentByNo(ctx context.Context, no string) (sqlcgen.Shipment, error) {
	return s.q.GetShipmentByNo(ctx, no)
}
