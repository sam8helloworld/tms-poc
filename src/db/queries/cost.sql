-- name: ListCostLineItemsByShipment :many
SELECT * FROM shipment_cost_line_items WHERE shipment_id = $1 ORDER BY cost_category, charge_code;

-- name: ListCostLineItemsByCategory :many
SELECT * FROM shipment_cost_line_items WHERE shipment_id = $1 AND cost_category = $2;

-- name: InsertCostLineItem :exec
INSERT INTO shipment_cost_line_items (
    id, shipment_id, cost_category, charge_code, charge_name, category,
    amount, currency, quantity, unit_price, unit_currency, applied_scope,
    remarks, segment_id, segment_index
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);

-- name: DeleteCostLineItemsByCategory :exec
DELETE FROM shipment_cost_line_items WHERE shipment_id = $1 AND cost_category = $2;

-- name: GetCostSummary :one
SELECT * FROM shipment_cost_summaries WHERE shipment_id = $1 AND cost_category = $2;

-- name: UpsertCostSummary :exec
INSERT INTO shipment_cost_summaries (
    id, shipment_id, cost_category, rate_id, invoice_id, invoice_no,
    provider_id, total_amount, total_currency, calculated_at, calculation_base
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (shipment_id, cost_category)
DO UPDATE SET
    rate_id = EXCLUDED.rate_id,
    invoice_id = EXCLUDED.invoice_id,
    invoice_no = EXCLUDED.invoice_no,
    provider_id = EXCLUDED.provider_id,
    total_amount = EXCLUDED.total_amount,
    total_currency = EXCLUDED.total_currency,
    calculated_at = EXCLUDED.calculated_at,
    calculation_base = EXCLUDED.calculation_base;
