-- name: GetContractByID :one
SELECT * FROM service_contracts WHERE id = $1;

-- name: ListContractsByProviderAndShipper :many
SELECT * FROM service_contracts WHERE provider_id = $1 AND shipper_id = $2 ORDER BY created_at DESC;

-- name: ListDraftContractsByProviderAndShipper :many
SELECT * FROM service_contracts
WHERE provider_id = $1 AND shipper_id = $2 AND status = 'DRAFT'
ORDER BY created_at DESC;

-- name: ListActiveContractsByProviderAndShipper :many
SELECT * FROM service_contracts
WHERE provider_id = $1 AND shipper_id = $2 AND status = 'CONTRACTED'
  AND valid_from <= $3 AND valid_to >= $3
ORDER BY created_at DESC;

-- name: ListContractsByShipper :many
SELECT * FROM service_contracts WHERE shipper_id = $1 ORDER BY created_at DESC;

-- name: InsertContract :exec
INSERT INTO service_contracts (id, provider_id, shipper_id, status, valid_from, valid_to, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateContract :exec
UPDATE service_contracts
SET status = $2, valid_from = $3, valid_to = $4, updated_at = $5
WHERE id = $1;

-- name: DeleteContract :exec
DELETE FROM service_contracts WHERE id = $1;
