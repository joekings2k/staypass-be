-- name: GetEstateBySlug :one
-- Get an estate by its slug.
SELECT * FROM estates WHERE slug = $1;

-- name: GetEstateByID :one
-- Get an estate by its ID.
SELECT * FROM estates WHERE id = $1;

-- name: ListEstates :many
-- List estates ordered by name, one page at a time.
SELECT * FROM estates
ORDER BY name
LIMIT @page_size OFFSET @page_offset;

-- name: CountEstates :one
-- Total number of estates, for computing the number of pages.
SELECT count(*) FROM estates;

-- name: CreateEstate :one
-- Create a new estate.
INSERT INTO estates (name, slug, address, city, state, timezone)
VALUES (@name, @slug, @address, @city, @state, @timezone)
RETURNING *;