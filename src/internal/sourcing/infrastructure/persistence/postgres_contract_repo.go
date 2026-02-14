package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
)

// PostgresContractRepo: ServiceContract集約のPostgreSQL実装
type PostgresContractRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresContractRepo(pool *pgxpool.Pool) *PostgresContractRepo {
	return &PostgresContractRepo{pool: pool}
}

func (r *PostgresContractRepo) Save(ctx context.Context, c *contract.ServiceContract) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO service_contracts (id, provider_id, shipper_id, status, valid_from, valid_to, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			valid_from = EXCLUDED.valid_from,
			valid_to = EXCLUDED.valid_to,
			updated_at = EXCLUDED.updated_at`,
		c.ID, c.ProviderID, c.ShipperID, string(c.Status()),
		c.ValidPeriod.From, c.ValidPeriod.To,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *PostgresContractRepo) FindByID(ctx context.Context, id uuid.UUID) (*contract.ServiceContract, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, provider_id, shipper_id, status, valid_from, valid_to, created_at, updated_at
		FROM service_contracts WHERE id = $1`, id)
	return scanContract(row)
}

func (r *PostgresContractRepo) FindByProviderAndShipper(ctx context.Context, providerID, shipperID uuid.UUID) ([]*contract.ServiceContract, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider_id, shipper_id, status, valid_from, valid_to, created_at, updated_at
		FROM service_contracts WHERE provider_id = $1 AND shipper_id = $2 ORDER BY created_at DESC`,
		providerID, shipperID)
	if err != nil {
		return nil, err
	}
	return collectContracts(rows)
}

func (r *PostgresContractRepo) FindDraftByProviderAndShipper(ctx context.Context, providerID, shipperID uuid.UUID) ([]*contract.ServiceContract, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider_id, shipper_id, status, valid_from, valid_to, created_at, updated_at
		FROM service_contracts WHERE provider_id = $1 AND shipper_id = $2 AND status = 'DRAFT'
		ORDER BY created_at DESC`, providerID, shipperID)
	if err != nil {
		return nil, err
	}
	return collectContracts(rows)
}

func (r *PostgresContractRepo) FindActiveByProviderAndShipper(ctx context.Context, providerID, shipperID uuid.UUID, effectiveDate time.Time) ([]*contract.ServiceContract, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider_id, shipper_id, status, valid_from, valid_to, created_at, updated_at
		FROM service_contracts
		WHERE provider_id = $1 AND shipper_id = $2 AND status = 'CONTRACTED'
			AND valid_from <= $3 AND valid_to >= $3
		ORDER BY created_at DESC`, providerID, shipperID, effectiveDate)
	if err != nil {
		return nil, err
	}
	return collectContracts(rows)
}

func (r *PostgresContractRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM service_contracts WHERE id = $1`, id)
	return err
}

func scanContract(s scannable) (*contract.ServiceContract, error) {
	var (
		id, providerID, shipperID uuid.UUID
		status                    string
		validFrom, validTo        time.Time
		createdAt, updatedAt      time.Time
	)
	err := s.Scan(&id, &providerID, &shipperID, &status, &validFrom, &validTo, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "contract not found")
		}
		return nil, err
	}

	return contract.ReconstructServiceContract(
		id, providerID, shipperID,
		contract.ContractStatus(status),
		shared.DateRange{From: validFrom, To: validTo},
		createdAt, updatedAt,
	), nil
}

func collectContracts(rows pgx.Rows) ([]*contract.ServiceContract, error) {
	defer rows.Close()
	var result []*contract.ServiceContract
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
