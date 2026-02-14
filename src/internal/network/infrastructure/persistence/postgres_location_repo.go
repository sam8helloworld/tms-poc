package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// PostgresLocationRepo: Location集約のPostgreSQL実装
type PostgresLocationRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresLocationRepo(pool *pgxpool.Pool) *PostgresLocationRepo {
	return &PostgresLocationRepo{pool: pool}
}

func (r *PostgresLocationRepo) Save(ctx context.Context, loc *route.Location) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO locations (id, name, un_locode, country_code, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, un_locode = EXCLUDED.un_locode,
			country_code = EXCLUDED.country_code, type = EXCLUDED.type,
			updated_at = NOW()`,
		uuid.UUID(loc.ID), loc.Name, loc.UnLocode, loc.CountryCode, loc.Type,
	)
	return err
}

func (r *PostgresLocationRepo) FindByID(ctx context.Context, id route.LocationID) (*route.Location, error) {
	var loc route.Location
	var locID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, un_locode, country_code, type
		FROM locations WHERE id = $1`, uuid.UUID(id)).
		Scan(&locID, &loc.Name, &loc.UnLocode, &loc.CountryCode, &loc.Type)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "location not found")
		}
		return nil, err
	}
	loc.ID = route.LocationID(locID)
	return &loc, nil
}

func (r *PostgresLocationRepo) FindByUnLocode(ctx context.Context, unLocode string) (*route.Location, error) {
	var loc route.Location
	var locID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, un_locode, country_code, type
		FROM locations WHERE un_locode = $1`, unLocode).
		Scan(&locID, &loc.Name, &loc.UnLocode, &loc.CountryCode, &loc.Type)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "location not found")
		}
		return nil, err
	}
	loc.ID = route.LocationID(locID)
	return &loc, nil
}
