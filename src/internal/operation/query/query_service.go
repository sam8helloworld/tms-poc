package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// OperationQueryService: Operation BCの読み取り専用クエリサービス
type OperationQueryService struct {
	q *sqlcgen.Queries
}

func NewOperationQueryService(pool *pgxpool.Pool) *OperationQueryService {
	return &OperationQueryService{q: sqlcgen.New(pool)}
}

func (s *OperationQueryService) GetSOPDefinition(ctx context.Context, id uuid.UUID) (sqlcgen.SopDefinition, error) {
	return s.q.GetSOPDefinitionByID(ctx, id)
}

func (s *OperationQueryService) ListSOPDefinitions(ctx context.Context) ([]sqlcgen.SopDefinition, error) {
	return s.q.ListActiveSOPDefinitions(ctx)
}

func (s *OperationQueryService) GetSOPInstance(ctx context.Context, id uuid.UUID) (sqlcgen.SopInstance, error) {
	return s.q.GetSOPInstanceByID(ctx, id)
}

func (s *OperationQueryService) GetSOPInstanceByShipment(ctx context.Context, shipmentID uuid.UUID) (sqlcgen.SopInstance, error) {
	return s.q.GetSOPInstanceByShipmentID(ctx, shipmentID)
}
