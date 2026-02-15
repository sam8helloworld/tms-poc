-- name: GetRateByID :one
SELECT * FROM rates WHERE id = $1;

-- name: ListActiveRatesByShipper :many
SELECT * FROM rates WHERE shipper_id = $1 AND status = 'ACTIVE' ORDER BY name;

-- name: InsertRate :exec
INSERT INTO rates (id, shipper_id, name, status, valid_from, valid_to, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateRate :exec
UPDATE rates SET name = $2, status = $3, valid_from = $4, valid_to = $5, updated_at = $6
WHERE id = $1;

-- name: DeleteRate :exec
DELETE FROM rates WHERE id = $1;

-- name: ListRateEntriesByRateID :many
SELECT * FROM rate_entries WHERE rate_id = $1;

-- name: InsertRateEntry :exec
INSERT INTO rate_entries (id, rate_id, provider_id, contract_id, tariff_id, origin_id, destination_id, transport_mode)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListRatesByShipper :many
SELECT * FROM rates WHERE shipper_id = $1 ORDER BY created_at DESC;

-- name: DeleteRateEntriesByRateID :exec
DELETE FROM rate_entries WHERE rate_id = $1;
