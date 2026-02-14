-- name: InsertDomainEvent :exec
INSERT INTO domain_events (id, event_type, aggregate_id, aggregate_type, payload, occurred_at, published, created_at)
VALUES ($1, $2, $3, $4, $5, $6, FALSE, NOW());

-- name: ListUnpublishedEvents :many
SELECT * FROM domain_events WHERE published = FALSE ORDER BY occurred_at LIMIT $1;

-- name: MarkEventPublished :exec
UPDATE domain_events SET published = TRUE, published_at = NOW() WHERE id = $1;

-- name: ListEventsByAggregate :many
SELECT * FROM domain_events WHERE aggregate_id = $1 AND aggregate_type = $2 ORDER BY occurred_at;
