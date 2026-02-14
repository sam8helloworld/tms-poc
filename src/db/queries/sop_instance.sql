-- name: GetSOPInstanceByID :one
SELECT * FROM sop_instances WHERE id = $1;

-- name: GetSOPInstanceByShipmentID :one
SELECT * FROM sop_instances WHERE shipment_id = $1;

-- name: InsertSOPInstance :exec
INSERT INTO sop_instances (id, shipment_id, definition_id, definition_name, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateSOPInstance :exec
UPDATE sop_instances SET status = $2, updated_at = $3 WHERE id = $1;

-- name: ListSOPTasksByInstanceID :many
SELECT * FROM sop_tasks WHERE sop_instance_id = $1 ORDER BY order_index;

-- name: InsertSOPTask :exec
INSERT INTO sop_tasks (
    id, sop_instance_id, step_definition_id, name, description,
    order_index, action_type, required_doc_types, generated_doc_type,
    status, assignee_id, linked_document_ids, completed_at, completed_by, note
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);

-- name: UpdateSOPTask :exec
UPDATE sop_tasks
SET status = $2, assignee_id = $3, linked_document_ids = $4,
    completed_at = $5, completed_by = $6, note = $7
WHERE id = $1;
