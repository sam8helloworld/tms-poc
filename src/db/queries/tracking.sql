-- name: GetTrackingUnitByID :one
SELECT * FROM tracking_units WHERE id = $1;

-- name: GetTrackingUnitByNumber :one
SELECT * FROM tracking_units WHERE tracking_number = $1;

-- name: InsertTrackingUnit :exec
INSERT INTO tracking_units (id, tracking_number, tracking_number_type, carrier_id, current_status, last_updated)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateTrackingUnit :exec
UPDATE tracking_units SET current_status = $2, last_updated = $3 WHERE id = $1;

-- name: ListTrackingSegmentsByUnitID :many
SELECT * FROM tracking_segments WHERE tracking_unit_id = $1;

-- name: InsertTrackingSegment :exec
INSERT INTO tracking_segments (
    id, tracking_unit_id, actual_origin_location_id, actual_dest_location_id,
    mode, carrier_tracking_number, primary_source, status,
    actual_departure, actual_arrival, estimated_arrival
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateTrackingSegment :exec
UPDATE tracking_segments
SET status = $2, actual_departure = $3, actual_arrival = $4, estimated_arrival = $5
WHERE id = $1;

-- name: DeleteTrackingSegmentsByUnitID :exec
DELETE FROM tracking_segments WHERE tracking_unit_id = $1;

-- name: ListTrackingEventsBySegmentID :many
SELECT * FROM tracking_events WHERE segment_id = $1 ORDER BY timestamp;

-- name: InsertTrackingEvent :exec
INSERT INTO tracking_events (id, segment_id, timestamp, source, code, description, location_raw, raw_payload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
