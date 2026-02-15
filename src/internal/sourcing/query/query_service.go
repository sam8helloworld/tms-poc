package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// SourcingQueryService: Sourcing BCの読み取り専用クエリサービス
type SourcingQueryService struct {
	q *sqlcgen.Queries
}

func NewSourcingQueryService(pool *pgxpool.Pool) *SourcingQueryService {
	return &SourcingQueryService{q: sqlcgen.New(pool)}
}

func (s *SourcingQueryService) GetContract(ctx context.Context, id uuid.UUID) (sqlcgen.ServiceContract, error) {
	return s.q.GetContractByID(ctx, id)
}

func (s *SourcingQueryService) ListContractsByShipper(ctx context.Context, shipperID uuid.UUID) ([]sqlcgen.ServiceContract, error) {
	return s.q.ListContractsByShipper(ctx, shipperID)
}

func (s *SourcingQueryService) GetTariff(ctx context.Context, id uuid.UUID) (sqlcgen.Tariff, error) {
	return s.q.GetTariffByID(ctx, id)
}

func (s *SourcingQueryService) ListTariffsByContract(ctx context.Context, contractID uuid.UUID) ([]sqlcgen.Tariff, error) {
	return s.q.ListTariffsByContractID(ctx, contractID)
}

func (s *SourcingQueryService) GetVendor(ctx context.Context, id uuid.UUID) (sqlcgen.Vendor, error) {
	return s.q.GetVendorByID(ctx, id)
}
