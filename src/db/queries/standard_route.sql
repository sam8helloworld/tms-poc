-- name: GetStandardRouteByID :one
SELECT * FROM standard_routes WHERE id = $1;

-- name: ListActiveStandardRoutesByShipper :many
SELECT * FROM standard_routes WHERE shipper_id = $1 AND status = 'ACTIVE' ORDER BY name;

-- name: InsertStandardRoute :exec
INSERT INTO standard_routes (
    id, name, shipper_id, origin_location_id, destination_location_id,
    status, standard_lead_time_days, target_cost_amount, target_cost_currency,
    valid_from, valid_to, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: UpdateStandardRoute :exec
UPDATE standard_routes
SET name = $2, status = $3, standard_lead_time_days = $4,
    target_cost_amount = $5, target_cost_currency = $6,
    valid_from = $7, valid_to = $8, updated_at = $9
WHERE id = $1;

-- name: ListStandardRouteLegsByRouteID :many
SELECT * FROM standard_route_legs WHERE standard_route_id = $1 ORDER BY sequence_order;

-- name: InsertStandardRouteLeg :exec
INSERT INTO standard_route_legs (
    id, standard_route_id, sequence_order, origin_location_id, dest_location_id,
    target_mode, standard_transit_days, master_lane_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: DeleteStandardRouteLegsByRouteID :exec
DELETE FROM standard_route_legs WHERE standard_route_id = $1;
