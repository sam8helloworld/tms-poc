package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// PostgresLaneRepo: Lane集約のPostgreSQL実装
type PostgresLaneRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresLaneRepo(pool *pgxpool.Pool) *PostgresLaneRepo {
	return &PostgresLaneRepo{pool: pool}
}

func (r *PostgresLaneRepo) Save(ctx context.Context, lane *route.Lane) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO lanes (id, origin_id, destination_id, mode, distance_km, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			mode = EXCLUDED.mode, distance_km = EXCLUDED.distance_km,
			updated_at = NOW()`,
		uuid.UUID(lane.ID), lane.OriginID, lane.DestinationID,
		string(lane.Mode), lane.DistanceKm,
	)
	return err
}

func (r *PostgresLaneRepo) FindByID(ctx context.Context, id route.LaneID) (*route.Lane, error) {
	var (
		laneID               uuid.UUID
		originID, destID     uuid.UUID
		mode                 string
		distanceKm           decimal.Decimal
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, origin_id, destination_id, mode, distance_km
		FROM lanes WHERE id = $1`, uuid.UUID(id)).
		Scan(&laneID, &originID, &destID, &mode, &distanceKm)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "lane not found")
		}
		return nil, err
	}
	return &route.Lane{
		ID:            route.LaneID(laneID),
		OriginID:      originID,
		DestinationID: destID,
		Mode:          shared.TransportMode(mode),
		DistanceKm:    distanceKm,
	}, nil
}

func (r *PostgresLaneRepo) FindByOriginAndDestination(ctx context.Context, originID, destID uuid.UUID) ([]*route.Lane, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, origin_id, destination_id, mode, distance_km
		FROM lanes WHERE origin_id = $1 AND destination_id = $2`, originID, destID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lanes []*route.Lane
	for rows.Next() {
		var (
			laneID       uuid.UUID
			oID, dID     uuid.UUID
			mode         string
			distKm       decimal.Decimal
		)
		if err := rows.Scan(&laneID, &oID, &dID, &mode, &distKm); err != nil {
			return nil, err
		}
		lanes = append(lanes, &route.Lane{
			ID:            route.LaneID(laneID),
			OriginID:      oID,
			DestinationID: dID,
			Mode:          shared.TransportMode(mode),
			DistanceKm:    distKm,
		})
	}
	return lanes, rows.Err()
}
