package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// PostgresRateRepo: Rate集約のPostgreSQL実装
type PostgresRateRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRateRepo(pool *pgxpool.Pool) *PostgresRateRepo {
	return &PostgresRateRepo{pool: pool}
}

func (r *PostgresRateRepo) Save(ctx context.Context, rt *rate.Rate) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO rates (id, shipper_id, name, status, valid_from, valid_to, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, status = EXCLUDED.status,
			valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to,
			updated_at = EXCLUDED.updated_at`,
		rt.ID, rt.ShipperID, rt.Name, string(rt.Status()),
		rt.ValidPeriod.From, rt.ValidPeriod.To,
		rt.CreatedAt, rt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert rate: %w", err)
	}

	// Replace entries
	_, err = tx.Exec(ctx, `DELETE FROM rate_entries WHERE rate_id = $1`, rt.ID)
	if err != nil {
		return fmt.Errorf("delete entries: %w", err)
	}

	for _, entry := range rt.Entries() {
		var originID, destID *uuid.UUID
		var mode *string
		if entry.RouteScope.OriginID != nil {
			id := uuid.UUID(*entry.RouteScope.OriginID)
			originID = &id
		}
		if entry.RouteScope.DestinationID != nil {
			id := uuid.UUID(*entry.RouteScope.DestinationID)
			destID = &id
		}
		if entry.RouteScope.TransportMode != nil {
			m := string(*entry.RouteScope.TransportMode)
			mode = &m
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO rate_entries (id, rate_id, provider_id, contract_id, tariff_id, origin_id, destination_id, transport_mode)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			entry.ID, rt.ID, entry.ProviderID, entry.ContractID, entry.TariffID,
			originID, destID, mode,
		)
		if err != nil {
			return fmt.Errorf("insert entry: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRateRepo) FindByID(ctx context.Context, id uuid.UUID) (*rate.Rate, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, shipper_id, name, status, valid_from, valid_to, created_at, updated_at
		FROM rates WHERE id = $1`, id)

	rt, err := scanRateHeader(row)
	if err != nil {
		return nil, err
	}

	entries, err := r.loadEntries(ctx, rt.ID)
	if err != nil {
		return nil, err
	}

	return rate.ReconstructRate(
		rt.ID, rt.ShipperID, rt.Name, rt.Status(),
		rt.ValidPeriod, entries, rt.CreatedAt, rt.UpdatedAt,
	), nil
}

func (r *PostgresRateRepo) FindActiveByShipper(ctx context.Context, shipperID uuid.UUID) ([]*rate.Rate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, shipper_id, name, status, valid_from, valid_to, created_at, updated_at
		FROM rates WHERE shipper_id = $1 AND status = 'ACTIVE' ORDER BY name`, shipperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []*rate.Rate
	for rows.Next() {
		rt, err := scanRateHeader(rows)
		if err != nil {
			return nil, err
		}
		entries, err := r.loadEntries(ctx, rt.ID)
		if err != nil {
			return nil, err
		}
		rates = append(rates, rate.ReconstructRate(
			rt.ID, rt.ShipperID, rt.Name, rt.Status(),
			rt.ValidPeriod, entries, rt.CreatedAt, rt.UpdatedAt,
		))
	}
	return rates, rows.Err()
}

func (r *PostgresRateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM rates WHERE id = $1`, id)
	return err
}

type scannable interface {
	Scan(dest ...any) error
}

// scanRateHeader scans a rate header row. Returns a temporary Rate for field access.
func scanRateHeader(s scannable) (*rate.Rate, error) {
	var (
		id, shipperID         uuid.UUID
		name, status          string
		validFrom, validTo    time.Time
		createdAt, updatedAt  time.Time
	)
	err := s.Scan(&id, &shipperID, &name, &status, &validFrom, &validTo, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "rate not found")
		}
		return nil, err
	}
	return rate.ReconstructRate(
		id, shipperID, name, rate.RateStatus(status),
		shared.DateRange{From: validFrom, To: validTo},
		nil, createdAt, updatedAt,
	), nil
}

func (r *PostgresRateRepo) loadEntries(ctx context.Context, rateID uuid.UUID) ([]*rate.RateEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider_id, contract_id, tariff_id, origin_id, destination_id, transport_mode
		FROM rate_entries WHERE rate_id = $1`, rateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*rate.RateEntry
	for rows.Next() {
		var (
			entry    rate.RateEntry
			originID *uuid.UUID
			destID   *uuid.UUID
			mode     *string
		)
		err := rows.Scan(&entry.ID, &entry.ProviderID, &entry.ContractID, &entry.TariffID,
			&originID, &destID, &mode)
		if err != nil {
			return nil, err
		}

		if originID != nil {
			lid := route.LocationID(*originID)
			entry.RouteScope.OriginID = &lid
		}
		if destID != nil {
			lid := route.LocationID(*destID)
			entry.RouteScope.DestinationID = &lid
		}
		if mode != nil {
			m := shared.TransportMode(*mode)
			entry.RouteScope.TransportMode = &m
		}

		entries = append(entries, &entry)
	}
	return entries, rows.Err()
}
