-- name: GetTariffByID :one
SELECT * FROM tariffs WHERE id = $1;

-- name: ListTariffsByContractID :many
SELECT * FROM tariffs WHERE contract_id = $1 ORDER BY name, version;

-- name: ListTariffsByContractIDAndName :many
SELECT * FROM tariffs WHERE contract_id = $1 AND name = $2 ORDER BY version;

-- name: GetLatestTariffVersionByContractIDAndName :one
SELECT * FROM tariffs WHERE contract_id = $1 AND name = $2 ORDER BY version DESC LIMIT 1;

-- name: CountTariffsByContractID :one
SELECT COUNT(*) FROM tariffs WHERE contract_id = $1;

-- name: InsertTariff :exec
INSERT INTO tariffs (id, contract_id, name, version, base_version_id, effective_from, effective_to, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateTariff :exec
UPDATE tariffs
SET name = $2, version = $3, base_version_id = $4,
    effective_from = $5, effective_to = $6, updated_at = $7
WHERE id = $1;

-- name: DeleteTariff :exec
DELETE FROM tariffs WHERE id = $1;

-- name: ListTariffLineItemsByTariffID :many
SELECT * FROM tariff_line_items WHERE tariff_id = $1;

-- name: InsertTariffLineItem :exec
INSERT INTO tariff_line_items (id, tariff_id, charge_code, category, scope, logic, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteTariffLineItemsByTariffID :exec
DELETE FROM tariff_line_items WHERE tariff_id = $1;
