-- name: GetShipmentByID :one
SELECT * FROM shipments WHERE id = $1;

-- name: GetShipmentByNo :one
SELECT * FROM shipments WHERE shipment_no = $1;

-- name: InsertShipment :exec
INSERT INTO shipments (
    id, shipment_no, shipper_id, consignee_id, status,
    standard_route_id, rate_id, origin_location_id, dest_location_id,
    transport_requirements, cost_is_finalized, cost_finalized_at,
    created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: UpdateShipment :exec
UPDATE shipments
SET status = $2, cost_is_finalized = $3, cost_finalized_at = $4, updated_at = $5
WHERE id = $1;

-- name: ListShipmentSegments :many
SELECT * FROM shipment_planned_route_segments WHERE shipment_id = $1 ORDER BY sequence_order;

-- name: InsertShipmentSegment :exec
INSERT INTO shipment_planned_route_segments (
    id, shipment_id, sequence_order, origin_location_id, origin_type,
    dest_location_id, dest_type, mode, distance_km, master_lane_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: DeleteShipmentSegments :exec
DELETE FROM shipment_planned_route_segments WHERE shipment_id = $1;

-- name: ListShipmentItems :many
SELECT * FROM shipment_items WHERE shipment_id = $1;

-- name: InsertShipmentItem :exec
INSERT INTO shipment_items (
    id, shipment_id, commodity, hs_code, quantity, weight_kg, volume_m3,
    package_type, loaded_on_tracking_id, attributes
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: DeleteShipmentItems :exec
DELETE FROM shipment_items WHERE shipment_id = $1;

-- name: ListShipmentTrackingUnitIDs :many
SELECT tracking_unit_id FROM shipment_tracking_units WHERE shipment_id = $1;

-- name: InsertShipmentTrackingUnit :exec
INSERT INTO shipment_tracking_units (shipment_id, tracking_unit_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteShipmentTrackingUnits :exec
DELETE FROM shipment_tracking_units WHERE shipment_id = $1;
