-- name: ListMilestonesByShipmentID :many
SELECT * FROM shipment_milestones WHERE shipment_id = $1 ORDER BY sequence;

-- name: InsertMilestone :exec
INSERT INTO shipment_milestones (
    id, shipment_id, milestone_type, occurred_at, recorded_at,
    source_document_id, source_doc_type, payload, sequence
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
