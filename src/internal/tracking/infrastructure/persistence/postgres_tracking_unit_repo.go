package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// PostgresTrackingUnitRepo: TrackingUnit集約のPostgreSQL実装
type PostgresTrackingUnitRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTrackingUnitRepo(pool *pgxpool.Pool) *PostgresTrackingUnitRepo {
	return &PostgresTrackingUnitRepo{pool: pool}
}

func (r *PostgresTrackingUnitRepo) Save(ctx context.Context, tu *tracking.TrackingUnit) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Upsert tracking unit header
	_, err = tx.Exec(ctx, `
		INSERT INTO tracking_units (id, tracking_number, tracking_number_type, carrier_id, current_status, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			current_status = EXCLUDED.current_status,
			last_updated = EXCLUDED.last_updated`,
		uuid.UUID(tu.ID), tu.TrackingNumber.Number, string(tu.TrackingNumber.TrackingNumberType),
		tu.CarrierID, string(tu.CurrentStatus()), tu.LastUpdated,
	)
	if err != nil {
		return fmt.Errorf("upsert tracking unit: %w", err)
	}

	// Delete and re-insert segments
	_, err = tx.Exec(ctx, `DELETE FROM tracking_segments WHERE tracking_unit_id = $1`, uuid.UUID(tu.ID))
	if err != nil {
		return fmt.Errorf("delete segments: %w", err)
	}

	for _, seg := range tu.Segments() {
		_, err = tx.Exec(ctx, `
			INSERT INTO tracking_segments (
				id, tracking_unit_id, actual_origin_location_id, actual_dest_location_id,
				mode, carrier_tracking_number, primary_source, status,
				actual_departure, actual_arrival, estimated_arrival
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			seg.ID, uuid.UUID(tu.ID), seg.ActualOriginLocationID, seg.ActualDestLocationID,
			string(seg.Mode), seg.CarrierTrackingNumber, string(seg.PrimarySource),
			string(seg.Status), seg.ActualDeparture, seg.ActualArrival, seg.EstimatedArrival,
		)
		if err != nil {
			return fmt.Errorf("insert segment: %w", err)
		}

		// Insert events for this segment
		for _, evt := range seg.Events {
			_, err = tx.Exec(ctx, `
				INSERT INTO tracking_events (id, segment_id, timestamp, source, code, description, location_raw, raw_payload)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (id) DO NOTHING`,
				evt.ID, seg.ID, evt.Timestamp, string(evt.Source),
				evt.Code, evt.Description, evt.LocationRaw, evt.RawPayload,
			)
			if err != nil {
				return fmt.Errorf("insert event: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresTrackingUnitRepo) FindByID(ctx context.Context, id uuid.UUID) (*tracking.TrackingUnit, error) {
	var (
		tuID                                   uuid.UUID
		number, numberType, status             string
		carrierID                              uuid.UUID
		lastUpdated                            time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, tracking_number, tracking_number_type, carrier_id, current_status, last_updated
		FROM tracking_units WHERE id = $1`, id).
		Scan(&tuID, &number, &numberType, &carrierID, &status, &lastUpdated)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "tracking unit not found")
		}
		return nil, err
	}

	segments, err := r.loadSegments(ctx, tuID)
	if err != nil {
		return nil, err
	}

	return tracking.ReconstructTrackingUnit(
		tracking.TrackingUnitID(tuID),
		tracking.TrackingNumber{Number: number, TrackingNumberType: tracking.TrackingNumberType(numberType)},
		carrierID,
		segments,
		shared.TrackingStatus(status),
		lastUpdated,
	), nil
}

func (r *PostgresTrackingUnitRepo) FindByTrackingNumber(ctx context.Context, number string, numberType tracking.TrackingNumberType) (*tracking.TrackingUnit, error) {
	var tuID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM tracking_units WHERE tracking_number = $1 AND tracking_number_type = $2`,
		number, string(numberType)).Scan(&tuID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "tracking unit not found")
		}
		return nil, err
	}
	return r.FindByID(ctx, tuID)
}

func (r *PostgresTrackingUnitRepo) loadSegments(ctx context.Context, trackingUnitID uuid.UUID) ([]*tracking.TrackingSegment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actual_origin_location_id, actual_dest_location_id,
			mode, carrier_tracking_number, primary_source, status,
			actual_departure, actual_arrival, estimated_arrival
		FROM tracking_segments WHERE tracking_unit_id = $1`, trackingUnitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []*tracking.TrackingSegment
	for rows.Next() {
		var (
			seg  tracking.TrackingSegment
			mode string
			src  string
			st   string
		)
		err := rows.Scan(
			&seg.ID, &seg.ActualOriginLocationID, &seg.ActualDestLocationID,
			&mode, &seg.CarrierTrackingNumber, &src, &st,
			&seg.ActualDeparture, &seg.ActualArrival, &seg.EstimatedArrival,
		)
		if err != nil {
			return nil, err
		}
		seg.Mode = shared.TransportMode(mode)
		seg.PrimarySource = tracking.TrackingSourceType(src)
		seg.Status = shared.TrackingStatus(st)

		// Load events
		events, err := r.loadEvents(ctx, seg.ID)
		if err != nil {
			return nil, err
		}
		seg.Events = events
		segments = append(segments, &seg)
	}
	return segments, rows.Err()
}

func (r *PostgresTrackingUnitRepo) loadEvents(ctx context.Context, segmentID uuid.UUID) ([]tracking.TrackingEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, timestamp, source, code, description, location_raw, raw_payload
		FROM tracking_events WHERE segment_id = $1 ORDER BY timestamp`, segmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []tracking.TrackingEvent
	for rows.Next() {
		var (
			evt tracking.TrackingEvent
			src string
		)
		err := rows.Scan(&evt.ID, &evt.Timestamp, &src, &evt.Code, &evt.Description, &evt.LocationRaw, &evt.RawPayload)
		if err != nil {
			return nil, err
		}
		evt.Source = tracking.TrackingSourceType(src)
		events = append(events, evt)
	}
	return events, rows.Err()
}
