package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// PostgresStandardRouteRepo: StandardRoute集約のPostgreSQL実装
type PostgresStandardRouteRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresStandardRouteRepo(pool *pgxpool.Pool) *PostgresStandardRouteRepo {
	return &PostgresStandardRouteRepo{pool: pool}
}

func (r *PostgresStandardRouteRepo) Save(ctx context.Context, sr *route.StandardRoute) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var costAmount *decimal.Decimal
	var costCurrency *string
	if sr.TargetCost != nil {
		costAmount = &sr.TargetCost.Amount
		costCurrency = &sr.TargetCost.Currency
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO standard_routes (
			id, name, shipper_id, origin_location_id, destination_location_id,
			status, standard_lead_time_days, target_cost_amount, target_cost_currency,
			valid_from, valid_to, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, status = EXCLUDED.status,
			standard_lead_time_days = EXCLUDED.standard_lead_time_days,
			target_cost_amount = EXCLUDED.target_cost_amount,
			target_cost_currency = EXCLUDED.target_cost_currency,
			valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to,
			updated_at = EXCLUDED.updated_at`,
		uuid.UUID(sr.ID), sr.Name, sr.ShipperID,
		uuid.UUID(sr.OriginLocationID), uuid.UUID(sr.DestinationLocationID),
		string(sr.Status()), sr.StandardLeadTimeDays,
		costAmount, costCurrency,
		sr.ValidPeriod.From, sr.ValidPeriod.To,
		sr.CreatedAt, sr.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert standard route: %w", err)
	}

	// Replace legs
	_, err = tx.Exec(ctx, `DELETE FROM standard_route_legs WHERE standard_route_id = $1`, uuid.UUID(sr.ID))
	if err != nil {
		return fmt.Errorf("delete legs: %w", err)
	}

	for _, leg := range sr.Legs() {
		var masterLaneID *uuid.UUID
		if leg.MasterLaneID != nil {
			id := uuid.UUID(*leg.MasterLaneID)
			masterLaneID = &id
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO standard_route_legs (
				id, standard_route_id, sequence_order, origin_location_id, dest_location_id,
				target_mode, standard_transit_days, master_lane_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			uuid.New(), uuid.UUID(sr.ID), leg.SequenceOrder,
			uuid.UUID(leg.OriginLocationID), uuid.UUID(leg.DestLocationID),
			string(leg.TargetMode), leg.StandardTransitDays, masterLaneID,
		)
		if err != nil {
			return fmt.Errorf("insert leg: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresStandardRouteRepo) FindByID(ctx context.Context, id route.StandardRouteID) (*route.StandardRoute, error) {
	var (
		srID, shipperID, originLocID, destLocID uuid.UUID
		name, status                            string
		leadTimeDays                            int
		costAmount                              *decimal.Decimal
		costCurrency                            *string
		validFrom, validTo                      time.Time
		createdAt, updatedAt                    time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, shipper_id, origin_location_id, destination_location_id,
			status, standard_lead_time_days, target_cost_amount, target_cost_currency,
			valid_from, valid_to, created_at, updated_at
		FROM standard_routes WHERE id = $1`, uuid.UUID(id)).
		Scan(&srID, &name, &shipperID, &originLocID, &destLocID,
			&status, &leadTimeDays, &costAmount, &costCurrency,
			&validFrom, &validTo, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "standard route not found")
		}
		return nil, err
	}

	legs, err := r.loadLegs(ctx, srID)
	if err != nil {
		return nil, err
	}

	var targetCost *shared.Money
	if costAmount != nil && costCurrency != nil {
		m := shared.Money{Amount: *costAmount, Currency: *costCurrency}
		targetCost = &m
	}

	return route.ReconstructStandardRoute(
		route.StandardRouteID(srID), name, shipperID,
		route.LocationID(originLocID), route.LocationID(destLocID),
		legs, route.StandardRouteStatus(status),
		leadTimeDays, targetCost,
		shared.DateRange{From: validFrom, To: validTo},
		createdAt, updatedAt,
	), nil
}

func (r *PostgresStandardRouteRepo) FindActiveByShipper(ctx context.Context, shipperID uuid.UUID) ([]*route.StandardRoute, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, shipper_id, origin_location_id, destination_location_id,
			status, standard_lead_time_days, target_cost_amount, target_cost_currency,
			valid_from, valid_to, created_at, updated_at
		FROM standard_routes WHERE shipper_id = $1 AND status = 'ACTIVE' ORDER BY name`, shipperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*route.StandardRoute
	for rows.Next() {
		var (
			srID, sID, oID, dID uuid.UUID
			name, status        string
			leadTime            int
			costAmt             *decimal.Decimal
			costCur             *string
			vFrom, vTo          time.Time
			cAt, uAt            time.Time
		)
		err := rows.Scan(&srID, &name, &sID, &oID, &dID,
			&status, &leadTime, &costAmt, &costCur,
			&vFrom, &vTo, &cAt, &uAt)
		if err != nil {
			return nil, err
		}

		legs, err := r.loadLegs(ctx, srID)
		if err != nil {
			return nil, err
		}

		var tc *shared.Money
		if costAmt != nil && costCur != nil {
			m := shared.Money{Amount: *costAmt, Currency: *costCur}
			tc = &m
		}

		routes = append(routes, route.ReconstructStandardRoute(
			route.StandardRouteID(srID), name, sID,
			route.LocationID(oID), route.LocationID(dID),
			legs, route.StandardRouteStatus(status),
			leadTime, tc,
			shared.DateRange{From: vFrom, To: vTo},
			cAt, uAt,
		))
	}
	return routes, rows.Err()
}

func (r *PostgresStandardRouteRepo) loadLegs(ctx context.Context, routeID uuid.UUID) ([]route.StandardRouteLeg, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sequence_order, origin_location_id, dest_location_id,
			target_mode, standard_transit_days, master_lane_id
		FROM standard_route_legs WHERE standard_route_id = $1 ORDER BY sequence_order`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var legs []route.StandardRouteLeg
	for rows.Next() {
		var (
			leg          route.StandardRouteLeg
			originID     uuid.UUID
			destID       uuid.UUID
			mode         string
			masterLaneID *uuid.UUID
		)
		err := rows.Scan(&leg.SequenceOrder, &originID, &destID, &mode, &leg.StandardTransitDays, &masterLaneID)
		if err != nil {
			return nil, err
		}
		leg.OriginLocationID = route.LocationID(originID)
		leg.DestLocationID = route.LocationID(destID)
		leg.TargetMode = shared.TransportMode(mode)
		if masterLaneID != nil {
			lid := route.LaneID(*masterLaneID)
			leg.MasterLaneID = &lid
		}
		legs = append(legs, leg)
	}
	return legs, rows.Err()
}
