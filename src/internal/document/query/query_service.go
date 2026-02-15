package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// DocumentQueryService: Document BCの読み取り専用クエリサービス
type DocumentQueryService struct {
	q *sqlcgen.Queries
}

func NewDocumentQueryService(pool *pgxpool.Pool) *DocumentQueryService {
	return &DocumentQueryService{q: sqlcgen.New(pool)}
}

func (s *DocumentQueryService) GetDocument(ctx context.Context, id uuid.UUID) (sqlcgen.Document, error) {
	return s.q.GetDocumentByID(ctx, id)
}

func (s *DocumentQueryService) ListDocumentsByShipment(ctx context.Context, shipmentID uuid.UUID) ([]sqlcgen.Document, error) {
	return s.q.ListDocumentsByShipmentID(ctx, shipmentID)
}
