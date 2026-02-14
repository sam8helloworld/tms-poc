-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = $1;

-- name: ListDocumentsByShipmentID :many
SELECT * FROM documents WHERE shipment_id = $1 ORDER BY created_at;

-- name: ListDocumentsByShipmentIDAndDocType :many
SELECT * FROM documents WHERE shipment_id = $1 AND doc_type = $2 ORDER BY version DESC;

-- name: InsertDocument :exec
INSERT INTO documents (
    id, shipment_id, doc_type, origin, file_name, mime_type,
    storage_uri, file_size, uploaded_by, status, version,
    metadata, content, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);

-- name: UpdateDocument :exec
UPDATE documents
SET status = $2, version = $3, metadata = $4, content = $5, updated_at = $6
WHERE id = $1;

-- name: ListDocumentReviewsByDocumentID :many
SELECT * FROM document_reviews WHERE document_id = $1 ORDER BY reviewed_at;

-- name: InsertDocumentReview :exec
INSERT INTO document_reviews (id, document_id, reviewer_id, reviewed_at, decision, comment, discrepancies)
VALUES ($1, $2, $3, $4, $5, $6, $7);
