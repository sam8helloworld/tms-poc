-- name: GetLaneByID :one
SELECT * FROM lanes WHERE id = $1;

-- name: GetLaneByOriginAndDestination :many
SELECT * FROM lanes WHERE origin_id = $1 AND destination_id = $2;

-- name: InsertLane :exec
INSERT INTO lanes (id, origin_id, destination_id, mode, distance_km, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateLane :exec
UPDATE lanes SET mode = $2, distance_km = $3, updated_at = $4 WHERE id = $1;
