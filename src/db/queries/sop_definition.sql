-- name: GetSOPDefinitionByID :one
SELECT * FROM sop_definitions WHERE id = $1;

-- name: ListActiveSOPDefinitions :many
SELECT * FROM sop_definitions WHERE status = 'ACTIVE' ORDER BY name;

-- name: ListActiveSOPDefinitionsByScope :many
SELECT * FROM sop_definitions
WHERE status = 'ACTIVE'
  AND direction = $1
  AND transport_mode = $2
  AND (origin_country IS NULL OR origin_country = $3)
  AND (dest_country IS NULL OR dest_country = $4)
ORDER BY name;

-- name: InsertSOPDefinition :exec
INSERT INTO sop_definitions (
    id, name, description, direction, transport_mode,
    origin_country, dest_country, status, version, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateSOPDefinition :exec
UPDATE sop_definitions
SET name = $2, description = $3, status = $4, version = $5, updated_at = $6
WHERE id = $1;

-- name: ListSOPStepsByDefinitionID :many
SELECT * FROM sop_step_definitions WHERE sop_definition_id = $1 ORDER BY order_index;

-- name: InsertSOPStep :exec
INSERT INTO sop_step_definitions (
    id, sop_definition_id, name, description, order_index,
    required_doc_types, generated_doc_type, action_type, is_automatable
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: DeleteSOPStepsByDefinitionID :exec
DELETE FROM sop_step_definitions WHERE sop_definition_id = $1;
