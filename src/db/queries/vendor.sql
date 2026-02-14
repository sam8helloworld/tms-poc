-- name: GetVendorByID :one
SELECT * FROM vendors WHERE id = $1;

-- name: ListVendorsByName :many
SELECT * FROM vendors WHERE name ILIKE '%' || $1 || '%' ORDER BY name;

-- name: InsertVendor :exec
INSERT INTO vendors (
    id, name, type, credit_rating, payment_days, payment_currency,
    preferred_vendor, capabilities, contacts, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateVendor :exec
UPDATE vendors
SET name = $2, type = $3, credit_rating = $4, payment_days = $5,
    payment_currency = $6, preferred_vendor = $7, capabilities = $8,
    contacts = $9, updated_at = $10
WHERE id = $1;
