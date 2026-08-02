-- name: GetHosts :one
SELECT * FROM hosts
WHERE host_id = $1 LIMIT 1;

-- name: ListHosts :many
SELECT * FROM hosts
ORDER BY name;
