package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/shipment/domain/shipment"
	"github.com/shopspring/decimal"
)

// PostgresShipmentRepo: Shipment集約のPostgreSQL実装
type PostgresShipmentRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresShipmentRepo(pool *pgxpool.Pool) *PostgresShipmentRepo {
	return &PostgresShipmentRepo{pool: pool}
}

func (r *PostgresShipmentRepo) Save(ctx context.Context, s *shipment.Shipment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	transportReqJSON, err := json.Marshal(s.Plan.TransportRequirements)
	if err != nil {
		return fmt.Errorf("marshal transport requirements: %w", err)
	}

	cost := s.Cost()

	_, err = tx.Exec(ctx, `
		INSERT INTO shipments (
			id, shipment_no, shipper_id, consignee_id, status,
			standard_route_id, rate_id, origin_location_id, dest_location_id,
			transport_requirements, cost_is_finalized, cost_finalized_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			cost_is_finalized = EXCLUDED.cost_is_finalized,
			cost_finalized_at = EXCLUDED.cost_finalized_at,
			updated_at = EXCLUDED.updated_at`,
		s.ID, s.ShipmentNo, s.ShipperID, s.ConsigneeID, string(s.Status()),
		s.Plan.StandardRouteID, s.Plan.RateID,
		uuid.UUID(s.Plan.PlannedRoute.OriginID), uuid.UUID(s.Plan.PlannedRoute.DestinationID),
		transportReqJSON, cost.IsFinalized, cost.FinalizedAt,
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert shipment: %w", err)
	}

	// Save planned route segments
	_, _ = tx.Exec(ctx, `DELETE FROM shipment_planned_route_segments WHERE shipment_id = $1`, s.ID)
	for _, seg := range s.Plan.PlannedRoute.Segments {
		var masterLaneID *uuid.UUID
		if seg.MasterLaneID != nil {
			id := uuid.UUID(*seg.MasterLaneID)
			masterLaneID = &id
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO shipment_planned_route_segments (
				id, shipment_id, sequence_order, origin_location_id, origin_type,
				dest_location_id, dest_type, mode, distance_km, master_lane_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			uuid.UUID(seg.ID), s.ID, seg.SequenceOrder,
			uuid.UUID(seg.OriginLocationID), string(seg.OriginType),
			uuid.UUID(seg.DestLocationID), string(seg.DestType),
			string(seg.Mode), seg.DistanceKm, masterLaneID,
		)
		if err != nil {
			return fmt.Errorf("insert segment: %w", err)
		}
	}

	// Save items
	_, _ = tx.Exec(ctx, `DELETE FROM shipment_items WHERE shipment_id = $1`, s.ID)
	for _, item := range s.Plan.Items {
		attrsJSON, err := json.Marshal(item.Attributes)
		if err != nil {
			return fmt.Errorf("marshal item attributes: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO shipment_items (
				id, shipment_id, commodity, hs_code, quantity, weight_kg, volume_m3,
				package_type, loaded_on_tracking_id, attributes
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			item.ID, s.ID, item.Commodity, item.HSCode,
			item.Quantity, item.WeightKG, item.VolumeM3,
			item.PackageType, item.LoadedOnTrackingID, attrsJSON,
		)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	// Save tracking unit references
	_, _ = tx.Exec(ctx, `DELETE FROM shipment_tracking_units WHERE shipment_id = $1`, s.ID)
	for _, tuID := range s.TrackingUnitIDs() {
		_, err = tx.Exec(ctx, `
			INSERT INTO shipment_tracking_units (shipment_id, tracking_unit_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, s.ID, tuID)
		if err != nil {
			return fmt.Errorf("insert tracking unit ref: %w", err)
		}
	}

	// Save milestones (append-only via ON CONFLICT DO NOTHING)
	for _, m := range s.Execution().Milestones() {
		payloadJSON, err := json.Marshal(m.Payload)
		if err != nil {
			return fmt.Errorf("marshal milestone payload: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO shipment_milestones (
				id, shipment_id, milestone_type, occurred_at, recorded_at,
				source_document_id, source_doc_type, payload, sequence
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING`,
			m.ID, s.ID, string(m.Type), m.OccurredAt, m.RecordedAt,
			m.SourceDocumentID, string(m.SourceDocType), payloadJSON, m.Sequence,
		)
		if err != nil {
			return fmt.Errorf("insert milestone: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresShipmentRepo) FindByID(ctx context.Context, id uuid.UUID) (*shipment.Shipment, error) {
	var (
		sID, shipperID, consigneeID uuid.UUID
		shipmentNo, status          string
		standardRouteID             *uuid.UUID
		rateID                      uuid.UUID
		originLocID, destLocID      uuid.UUID
		transportReqJSON            []byte
		costFinalized               bool
		costFinalizedAt             *time.Time
		createdAt, updatedAt        time.Time
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, shipment_no, shipper_id, consignee_id, status,
			standard_route_id, rate_id, origin_location_id, dest_location_id,
			transport_requirements, cost_is_finalized, cost_finalized_at,
			created_at, updated_at
		FROM shipments WHERE id = $1`, id).
		Scan(&sID, &shipmentNo, &shipperID, &consigneeID, &status,
			&standardRouteID, &rateID, &originLocID, &destLocID,
			&transportReqJSON, &costFinalized, &costFinalizedAt,
			&createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "shipment not found")
		}
		return nil, err
	}

	// Load segments
	segments, err := r.loadSegments(ctx, sID)
	if err != nil {
		return nil, err
	}

	// Load items
	items, err := r.loadItems(ctx, sID)
	if err != nil {
		return nil, err
	}

	// Load tracking unit IDs
	trackingUnitIDs, err := r.loadTrackingUnitIDs(ctx, sID)
	if err != nil {
		return nil, err
	}

	// Load milestones
	milestones, err := r.loadMilestones(ctx, sID)
	if err != nil {
		return nil, err
	}

	var transportReq map[string]interface{}
	if transportReqJSON != nil {
		_ = json.Unmarshal(transportReqJSON, &transportReq)
	}

	plan := shipment.ShipmentPlan{
		StandardRouteID: standardRouteID,
		PlannedRoute: route.PhysicalRoute{
			ID:            route.PhysicalRouteID(sID),
			OriginID:      route.LocationID(originLocID),
			DestinationID: route.LocationID(destLocID),
			Segments:      segments,
		},
		Items:                 items,
		RateID:                rateID,
		TransportRequirements: transportReq,
	}

	cost := shipment.ShipmentCost{
		IsFinalized: costFinalized,
		FinalizedAt: costFinalizedAt,
	}

	execution := shipment.ReconstructShipmentExecution(milestones)

	return shipment.ReconstructShipment(
		sID, shipmentNo, shipperID, consigneeID,
		shipment.ShipmentStatus(status),
		plan, execution, trackingUnitIDs, cost,
		createdAt, updatedAt,
	), nil
}

func (r *PostgresShipmentRepo) loadSegments(ctx context.Context, shipmentID uuid.UUID) ([]route.RouteSegment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sequence_order, origin_location_id, origin_type,
			dest_location_id, dest_type, mode, distance_km, master_lane_id
		FROM shipment_planned_route_segments WHERE shipment_id = $1 ORDER BY sequence_order`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []route.RouteSegment
	for rows.Next() {
		var (
			seg          route.RouteSegment
			segID        uuid.UUID
			originID     uuid.UUID
			destID       uuid.UUID
			originType   string
			destType     string
			mode         string
			distanceKm   decimal.Decimal
			masterLaneID *uuid.UUID
		)
		err := rows.Scan(
			&segID, &seg.SequenceOrder, &originID, &originType,
			&destID, &destType, &mode, &distanceKm, &masterLaneID,
		)
		if err != nil {
			return nil, err
		}
		seg.ID = route.RouteSegmentID(segID)
		seg.OriginLocationID = route.LocationID(originID)
		seg.OriginType = shared.LocationType(originType)
		seg.DestLocationID = route.LocationID(destID)
		seg.DestType = shared.LocationType(destType)
		seg.Mode = shared.TransportMode(mode)
		seg.DistanceKm = distanceKm
		if masterLaneID != nil {
			lid := route.LaneID(*masterLaneID)
			seg.MasterLaneID = &lid
		}
		segments = append(segments, seg)
	}
	return segments, rows.Err()
}

func (r *PostgresShipmentRepo) loadItems(ctx context.Context, shipmentID uuid.UUID) ([]shipment.ShipmentItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, commodity, hs_code, quantity, weight_kg, volume_m3,
			package_type, loaded_on_tracking_id, attributes
		FROM shipment_items WHERE shipment_id = $1`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []shipment.ShipmentItem
	for rows.Next() {
		var (
			item     shipment.ShipmentItem
			attrsJSON []byte
		)
		err := rows.Scan(
			&item.ID, &item.Commodity, &item.HSCode,
			&item.Quantity, &item.WeightKG, &item.VolumeM3,
			&item.PackageType, &item.LoadedOnTrackingID, &attrsJSON,
		)
		if err != nil {
			return nil, err
		}
		if attrsJSON != nil {
			_ = json.Unmarshal(attrsJSON, &item.Attributes)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresShipmentRepo) loadTrackingUnitIDs(ctx context.Context, shipmentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tracking_unit_id FROM shipment_tracking_units WHERE shipment_id = $1`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresShipmentRepo) loadMilestones(ctx context.Context, shipmentID uuid.UUID) ([]shipment.Milestone, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, milestone_type, occurred_at, recorded_at,
			source_document_id, source_doc_type, payload, sequence
		FROM shipment_milestones WHERE shipment_id = $1 ORDER BY sequence`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var milestones []shipment.Milestone
	for rows.Next() {
		var (
			m           shipment.Milestone
			mType       string
			srcDocType  string
			payloadJSON []byte
		)
		err := rows.Scan(
			&m.ID, &mType, &m.OccurredAt, &m.RecordedAt,
			&m.SourceDocumentID, &srcDocType, &payloadJSON, &m.Sequence,
		)
		if err != nil {
			return nil, err
		}
		m.Type = shipment.MilestoneType(mType)
		m.SourceDocType = shared.DocType(srcDocType)
		// Note: Payload deserialization requires polymorphic mapping
		// For now set a GenericPayload with raw data
		var rawData map[string]interface{}
		if payloadJSON != nil {
			_ = json.Unmarshal(payloadJSON, &rawData)
		}
		m.Payload = shipment.GenericPayload{MType: m.Type, Data: rawData}

		milestones = append(milestones, m)
	}
	return milestones, rows.Err()
}
