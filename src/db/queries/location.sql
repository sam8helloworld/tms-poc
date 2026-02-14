-- name: GetLocationByID :one
SELECT * FROM locations WHERE id = $1;

-- name: GetLocationByUnLocode :one
SELECT * FROM locations WHERE un_locode = $1;

-- name: ListLocationsByCountry :many
SELECT * FROM locations WHERE country_code = $1 ORDER BY name;

-- name: InsertLocation :exec
INSERT INTO locations (id, name, un_locode, country_code, type, attributes, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateLocation :exec
UPDATE locations
SET name = $2, un_locode = $3, country_code = $4, type = $5, attributes = $6, updated_at = $7
WHERE id = $1;
